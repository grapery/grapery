// Command seed inserts ~100–200 rows of local test data (users, follows, stories,
// storyboards with child nodes, fragments, comments). Safe to re-run: exits if seed
// users already exist unless -force is passed.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm"

	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
)

const (
	seedUsernamePrefix = "seed_user_"
	defaultPassword    = "test123456"
)

// 碎片话题标签（与广场 `fragments_topic` 示例一致含「日常」）；非故事「体裁」。
var fragmentTopicTags = []string{"日常", "灵感", "连载", "随笔", "幕后"}

func main() {
	force := flag.Bool("force", false, "delete existing seed users (seed_user_*) and related rows, then re-seed")
	flag.Parse()

	_ = godotenv.Load()

	log, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	cfg := config.Load("seed")
	if cfg.Database.Password == "" {
		log.Fatal("DB_PASSWORD (or config) must be set for seed")
	}

	db, err := mysql.InitDB(cfg.Database.DSN(), log)
	if err != nil {
		log.Fatal("database", zap.Error(err))
	}

	repo := mysql.NewRepository(db, log, cfg.Recommendation)
	fragInteract := repository.NewFragmentInteractionRepository(repo.DB())
	ctx := context.Background()

	var existing int64
	if err := repo.DB().WithContext(ctx).Model(&mysql.User{}).
		Where("username LIKE ?", seedUsernamePrefix+"%").
		Count(&existing).Error; err != nil {
		log.Fatal("count seed users", zap.Error(err))
	}
	if existing > 0 && !*force {
		log.Fatal(fmt.Sprintf("found %d seed users (username like %q); use -force to remove and re-seed", existing, seedUsernamePrefix+"%"))
	}

	if *force && existing > 0 {
		if err := wipeSeedData(ctx, repo); err != nil {
			log.Fatal("wipe seed data", zap.Error(err))
		}
		log.Info("removed previous seed data")
	}

	userIDs, err := createUsers(ctx, repo, log, 15)
	if err != nil {
		log.Fatal("create users", zap.Error(err))
	}

	if err := createFollows(ctx, repo, userIDs); err != nil {
		log.Fatal("follows", zap.Error(err))
	}

	storyIDs, err := createStories(ctx, repo, userIDs, 15)
	if err != nil {
		log.Fatal("stories", zap.Error(err))
	}

	storyboardIDs, err := createStoryboards(ctx, repo, storyIDs, userIDs)
	if err != nil {
		log.Fatal("storyboards", zap.Error(err))
	}

	fragmentIDs, err := createFragments(ctx, repo, userIDs, 35)
	if err != nil {
		log.Fatal("fragments", zap.Error(err))
	}

	if err := createStoryComments(ctx, repo, storyIDs, storyboardIDs, userIDs); err != nil {
		log.Fatal("story comments", zap.Error(err))
	}

	if err := createFragmentComments(ctx, fragInteract, fragmentIDs, userIDs); err != nil {
		log.Fatal("fragment comments", zap.Error(err))
	}

	log.Info("seed completed",
		zap.Int("users", len(userIDs)),
		zap.Int("stories", len(storyIDs)),
		zap.Int("storyboards", len(storyboardIDs)),
		zap.Int("fragments", len(fragmentIDs)),
		zap.String("password", defaultPassword),
		zap.String("sample_login", seedUsernamePrefix+"01"),
	)
}

func wipeSeedData(ctx context.Context, repo *mysql.Repository) error {
	gdb := repo.DB().WithContext(ctx)

	var userIDs []string
	if err := gdb.Model(&mysql.User{}).Unscoped().
		Where("username LIKE ?", seedUsernamePrefix+"%").
		Pluck("id", &userIDs).Error; err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}

	var storyIDs []string
	if err := gdb.Model(&mysql.Story{}).Unscoped().
		Where("author_id IN ?", userIDs).
		Pluck("id", &storyIDs).Error; err != nil {
		return err
	}

	var storyboardIDs []string
	if len(storyIDs) > 0 {
		if err := gdb.Model(&mysql.Storyboard{}).Unscoped().
			Where("story_id IN ?", storyIDs).
			Pluck("id", &storyboardIDs).Error; err != nil {
			return err
		}
	}

	var fragmentIDs []string
	if err := gdb.Model(&mysql.FragmentDB{}).Unscoped().
		Where("creator_id IN ?", userIDs).
		Pluck("id", &fragmentIDs).Error; err != nil {
		return err
	}

	return gdb.Transaction(func(tx *gorm.DB) error {
		d := tx.Unscoped()
		if len(fragmentIDs) > 0 {
			if err := d.Where("fragment_id IN ?", fragmentIDs).Delete(&mysql.FragmentLikeDB{}).Error; err != nil {
				return err
			}
			if err := d.Where("fragment_id IN ?", fragmentIDs).Delete(&mysql.FragmentCommentDB{}).Error; err != nil {
				return err
			}
			if err := d.Where("id IN ?", fragmentIDs).Delete(&mysql.FragmentDB{}).Error; err != nil {
				return err
			}
		}
		if len(storyboardIDs) > 0 {
			if err := d.Where("target_type = ? AND target_id IN ?", "storyboard", storyboardIDs).Delete(&mysql.Comment{}).Error; err != nil {
				return err
			}
			if err := d.Where("storyboard_id IN ?", storyboardIDs).Delete(&mysql.StoryboardCharacterLink{}).Error; err != nil {
				return err
			}
			if err := d.Where("storyboard_id IN ?", storyboardIDs).Delete(&mysql.StoryboardSceneLink{}).Error; err != nil {
				return err
			}
			if err := d.Where("id IN ?", storyboardIDs).Delete(&mysql.Storyboard{}).Error; err != nil {
				return err
			}
		}
		if len(storyIDs) > 0 {
			if err := d.Where("target_type = ? AND target_id IN ?", "story", storyIDs).Delete(&mysql.Comment{}).Error; err != nil {
				return err
			}
			if err := d.Where("id IN ?", storyIDs).Delete(&mysql.Story{}).Error; err != nil {
				return err
			}
		}
		if err := d.Where("follower_id IN ? OR followee_id IN ?", userIDs, userIDs).Delete(&mysql.UserFollow{}).Error; err != nil {
			return err
		}
		return d.Where("id IN ?", userIDs).Delete(&mysql.User{}).Error
	})
}

func createUsers(ctx context.Context, repo *mysql.Repository, log *zap.Logger, n int) ([]string, error) {
	hash, err := authPkg.HashPassword(defaultPassword)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		uid := uuid.NewString()
		uname := fmt.Sprintf("%s%02d", seedUsernamePrefix, i)
		email := fmt.Sprintf("%s@seed.grapery.local", uname)
		refCode := fmt.Sprintf("S%03d%014d", i, i*100000) // unique, max 20 chars
		if len(refCode) > 20 {
			refCode = refCode[:20]
		}
		u := &domain.User{
			BaseModel:    common.BaseModel{ID: uid},
			Username:     uname,
			Email:        email,
			PasswordHash: hash,
			DisplayName:  fmt.Sprintf("Seed User %02d", i),
			Bio:          "Local seed account for development.",
			Status:       "active",
			EmailVerified: true,
			ReferralCode: refCode,
		}
		if err := repo.CreateUser(ctx, u); err != nil {
			return nil, fmt.Errorf("user %s: %w", uname, err)
		}
		ids = append(ids, uid)
		log.Info("created user", zap.String("username", uname))
	}
	return ids, nil
}

func createFollows(ctx context.Context, repo *mysql.Repository, userIDs []string) error {
	n := len(userIDs)
	for i, follower := range userIDs {
		for step := 1; step <= 3; step++ {
			j := (i + step) % n
			followee := userIDs[j]
			if follower == followee {
				continue
			}
			if err := repo.FollowUser(ctx, follower, followee); err != nil {
				return fmt.Errorf("follow %s -> %s: %w", follower, followee, err)
			}
		}
	}
	return nil
}

func createStories(ctx context.Context, repo *mysql.Repository, userIDs []string, count int) ([]string, error) {
	ids := make([]string, 0, count)
	for s := 0; s < count; s++ {
		authorID := userIDs[s%len(userIDs)]
		sid := uuid.NewString()
		st := &domain.Story{
			BaseModel:           common.BaseModel{ID: sid},
			Title:               fmt.Sprintf("种子故事 %d", s+1),
			Description:         fmt.Sprintf("本地种子数据：第 %d 条故事，用于联调列表与详情。", s+1),
			Genre:               "fiction",
			Status:              "published",
			Visibility:          "public",
			UseAI:               false,
			DefaultSceneCount:   3,
			IsCollaborationOpen: false,
			Author:              &domain.User{BaseModel: common.BaseModel{ID: authorID}},
		}
		if err := repo.CreateStory(ctx, st); err != nil {
			return nil, fmt.Errorf("story %d: %w", s, err)
		}
		ids = append(ids, sid)
	}
	return ids, nil
}

func createStoryboards(ctx context.Context, repo *mysql.Repository, storyIDs, userIDs []string) ([]string, error) {
	var all []string
	for idx, storyID := range storyIDs {
		creator := userIDs[idx%len(userIDs)]
		root := &domain.Storyboard{
			StoryID:        storyID,
			ParentID:       domain.StoryboardRootMarker,
			UserID:         creator,
			Title:          fmt.Sprintf("根故事板 %d", idx+1),
			Content:        fmt.Sprintf("根节点剧情概要 #%d，用于树状结构展示。", idx+1),
			RawInput:       fmt.Sprintf("seed raw input root %d", idx+1),
			WorkflowStatus: "published",
			SceneCount:     3,
			CurrentStep:    5,
		}
		if err := repo.CreateStoryboard(ctx, root); err != nil {
			return nil, fmt.Errorf("root sb story %s: %w", storyID, err)
		}
		if err := repo.IncrementStoryStoryboardCount(ctx, storyID); err != nil {
			return nil, err
		}
		if err := bumpUserStoryboardCount(ctx, repo, creator); err != nil {
			return nil, err
		}
		all = append(all, root.ID)

		child1 := &domain.Storyboard{
			StoryID:        storyID,
			ParentID:       root.ID,
			UserID:         creator,
			Title:          fmt.Sprintf("子故事板 %d-A", idx+1),
			Content:        "子节点 A：延续主线的一小段情节。",
			RawInput:       fmt.Sprintf("seed child A %d", idx+1),
			WorkflowStatus: "published",
			SceneCount:     3,
			CurrentStep:    5,
		}
		if err := repo.CreateStoryboard(ctx, child1); err != nil {
			return nil, fmt.Errorf("child1 sb: %w", err)
		}
		if err := repo.IncrementStoryStoryboardCount(ctx, storyID); err != nil {
			return nil, err
		}
		if err := bumpUserStoryboardCount(ctx, repo, creator); err != nil {
			return nil, err
		}
		all = append(all, child1.ID)

		child2 := &domain.Storyboard{
			StoryID:        storyID,
			ParentID:       child1.ID,
			UserID:         userIDs[(idx+1)%len(userIDs)],
			Title:          fmt.Sprintf("子故事板 %d-B（更深）", idx+1),
			Content:        "子节点 B：挂在 A 下的更深分支。",
			RawInput:       fmt.Sprintf("seed child B %d", idx+1),
			WorkflowStatus: "content_ready",
			SceneCount:     3,
			CurrentStep:    2,
		}
		if err := repo.CreateStoryboard(ctx, child2); err != nil {
			return nil, fmt.Errorf("child2 sb: %w", err)
		}
		if err := repo.IncrementStoryStoryboardCount(ctx, storyID); err != nil {
			return nil, err
		}
		if err := bumpUserStoryboardCount(ctx, repo, child2.UserID); err != nil {
			return nil, err
		}
		all = append(all, child2.ID)
	}
	return all, nil
}

func bumpUserStoryboardCount(ctx context.Context, repo *mysql.Repository, userID string) error {
	return repo.DB().WithContext(ctx).Model(&mysql.User{}).
		Where("id = ?", userID).
		UpdateColumn("storyboard_count", gorm.Expr("storyboard_count + ?", 1)).Error
}

func bumpUserFragmentsCount(ctx context.Context, repo *mysql.Repository, userID string) error {
	return repo.DB().WithContext(ctx).Model(&mysql.User{}).
		Where("id = ?", userID).
		UpdateColumn("fragments_count", gorm.Expr("fragments_count + ?", 1)).Error
}

func createFragments(ctx context.Context, repo *mysql.Repository, userIDs []string, n int) ([]string, error) {
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		author := userIDs[i%len(userIDs)]
		fid := uuid.NewString()
		topic := fragmentTopicTags[i%len(fragmentTopicTags)]
		isDraft := i%7 == 0
		vis := domain.FragmentVisibilityPublic
		if i%11 == 0 {
			vis = domain.FragmentVisibilityFollowers
		}
		frag := &domain.Fragment{
			BaseModel:   common.BaseModel{ID: fid},
			UserID:      author,
			Content:     fmt.Sprintf("【%s】种子碎片 #%d：用于广场与详情联调。", topic, i+1),
			Visibility:  vis,
			SourceType:  string(domain.FragmentSourceOriginal),
			Topic:       topic,
			Caption:     fmt.Sprintf("碎片标题 %d", i+1),
			IsDraft:     isDraft,
			MediaURLs:   nil,
		}
		if err := repo.CreateFragment(ctx, frag); err != nil {
			return nil, fmt.Errorf("fragment %d: %w", i, err)
		}
		if !isDraft {
			if err := bumpUserFragmentsCount(ctx, repo, author); err != nil {
				return nil, err
			}
		}
		ids = append(ids, fid)
	}
	return ids, nil
}

func createStoryComments(ctx context.Context, repo *mysql.Repository, storyIDs, storyboardIDs, userIDs []string) error {
	// Top-level and one reply on stories and storyboards
	for i, sid := range storyIDs {
		author := userIDs[i%len(userIDs)]
		c1 := &domain.Comment{
			UserID:     author,
			Content:    fmt.Sprintf("对故事的评论 #%d", i+1),
			TargetType: "story",
			TargetID:   sid,
		}
		if err := repo.CreateComment(ctx, c1); err != nil {
			return err
		}
		time.Sleep(2 * time.Millisecond) // avoid identical CreatedAt if UI sorts by time
		replier := userIDs[(i+1)%len(userIDs)]
		c2 := &domain.Comment{
			UserID:     replier,
			Content:    "回复：同意，情节很有趣。",
			TargetType: "story",
			TargetID:   sid,
			ParentID:   c1.ID,
		}
		if err := repo.CreateComment(ctx, c2); err != nil {
			return err
		}
	}

	for i := 0; i < len(storyboardIDs) && i < 12; i++ {
		sbid := storyboardIDs[i]
		author := userIDs[(i+2)%len(userIDs)]
		c := &domain.Comment{
			UserID:     author,
			Content:    fmt.Sprintf("故事板讨论 #%d", i+1),
			TargetType: "storyboard",
			TargetID:   sbid,
		}
		if err := repo.CreateComment(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func createFragmentComments(ctx context.Context, inter *repository.FragmentInteractionRepository, fragmentIDs, userIDs []string) error {
	for i := 0; i < len(fragmentIDs); i += 2 {
		fid := fragmentIDs[i]
		author := userIDs[i%len(userIDs)]
		parentID := uuid.NewString()
		top := &domain.FragmentComment{
			ID:         parentID,
			FragmentID: fid,
			UserID:     author,
			Content:    fmt.Sprintf("碎片评论 #%d", i+1),
		}
		if err := inter.CreateComment(ctx, top); err != nil {
			return err
		}
		if i+1 >= len(userIDs) {
			continue
		}
		pid := parentID
		reply := &domain.FragmentComment{
			ID:         uuid.NewString(),
			FragmentID: fid,
			UserID:     userIDs[(i+3)%len(userIDs)],
			Content:    "回复：这条碎片不错。",
			ParentID:   &pid,
		}
		if err := inter.CreateComment(ctx, reply); err != nil {
			return err
		}
	}
	return nil
}
