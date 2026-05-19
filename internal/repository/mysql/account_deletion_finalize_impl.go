package mysql

import (
	"context"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ApplyAccountDeletionContentReassignment moves remaining public authored rows to the system placeholder user,
// rewires polymorphic authors on comments/replies for that user ID, and refreshes aggregated counts.
func (r *Repository) ApplyAccountDeletionContentReassignment(ctx context.Context, fromUID, sysUID string) error {
	fromUID = strings.TrimSpace(fromUID)
	sysUID = strings.TrimSpace(sysUID)
	if fromUID == "" || sysUID == "" || fromUID == sysUID {
		return domain.ErrInvalidInput
	}

	pub := string(common.ContentStatusPublished)
	pubVis := string(domain.StoryVisibilityPublic)
	frPub := domain.FragmentVisibilityPublic

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		st := tx.Exec(`
			UPDATE stories
			SET author_id = ?
			WHERE author_id = ? AND status = ? AND visibility = ? AND deleted_at IS NULL`,
			sysUID, fromUID, pub, pubVis)
		if st.Error != nil {
			return st.Error
		}

		sb := tx.Exec(`
			UPDATE storyboards sb
			INNER JOIN stories s ON s.id = sb.story_id
			SET sb.creator_id = ?
			WHERE sb.creator_id = ?
			  AND sb.deleted_at IS NULL
			  AND s.deleted_at IS NULL
			  AND s.status = ?
			  AND s.visibility = ?`,
			sysUID, fromUID, pub, pubVis)
		if sb.Error != nil {
			return sb.Error
		}

		fr := tx.Exec(`
			UPDATE fragments
			SET creator_id = ?
			WHERE creator_id = ?
			  AND COALESCE(is_draft, 0) = 0
			  AND visibility = ?`,
			sysUID, fromUID, frPub)
		if fr.Error != nil {
			return fr.Error
		}

		ch := tx.Exec(`
			UPDATE characters c
			INNER JOIN stories s ON s.id = c.story_id
			SET c.author_id = ?
			WHERE c.author_id = ?
			  AND c.deleted_at IS NULL
			  AND s.deleted_at IS NULL
			  AND s.status = ?
			  AND s.visibility = ?`,
			sysUID, fromUID, pub, pubVis)
		if ch.Error != nil {
			return ch.Error
		}

		if err := tx.Exec("UPDATE comments SET author_id = ? WHERE author_id = ?", sysUID, fromUID).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE fragment_comments SET user_id = ? WHERE user_id = ?", sysUID, fromUID).Error; err != nil {
			return err
		}
		// Best-effort: actor linkage on persisted notifications only.
		_ = tx.Exec("UPDATE notifications SET actor_id = ? WHERE actor_id = ?", sysUID, fromUID).Error

		var fc int64
		if err := tx.Model(&FragmentDB{}).Where("creator_id = ?", sysUID).Count(&fc).Error; err != nil {
			return err
		}
		var sbc int64
		if err := tx.Model(&Storyboard{}).
			Where("creator_id = ? AND deleted_at IS NULL", sysUID).
			Count(&sbc).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", sysUID).Updates(map[string]interface{}{
			"fragments_count":  fc,
			"storyboard_count": sbc,
		}).Error; err != nil {
			return err
		}

		var leftFr int64
		if err := tx.Model(&FragmentDB{}).Where("creator_id = ?", fromUID).Count(&leftFr).Error; err != nil {
			return err
		}
		var leftSb int64
		if err := tx.Model(&Storyboard{}).
			Where("creator_id = ? AND deleted_at IS NULL", fromUID).
			Count(&leftSb).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", fromUID).Updates(map[string]interface{}{
			"fragments_count":  leftFr,
			"storyboard_count": leftSb,
		}).Error; err != nil {
			return err
		}
		return nil
	})
}

// ApplyAccountDeletionUserSocialGraphPurge deletes follows, bookmarks, likes, saves, notifications, and settings tied to userID.
func (r *Repository) ApplyAccountDeletionUserSocialGraphPurge(ctx context.Context, uid string) error {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return domain.ErrInvalidInput
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		fn := tx.Session(&gorm.Session{SkipHooks: true}).Unscoped()

		if err := fn.Where("user_id = ?", uid).Delete(&Notification{}).Error; err != nil {
			return err
		}
		if err := fn.Where("follower_id = ? OR followee_id = ?", uid, uid).Delete(&UserFollow{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&StoryFollow{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&StoryLike{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&CharacterFollow{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&StoryboardLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&Bookmark{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&Like{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&CommentLike{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&FragmentLikeDB{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&FragmentShareDB{}).Error; err != nil {
			return err
		}

		if err := fn.Where("blocker_id = ? OR blocked_id = ?", uid, uid).Delete(&UserBlock{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&UserFeedback{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&UserSettings{}).Error; err != nil {
			return err
		}
		if err := fn.Where("user_id = ?", uid).Delete(&CharacterGenerationTask{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", uid).Delete(&FragmentGenerationTaskDB{}).Error; err != nil {
			return err
		}
		return nil
	})
}
