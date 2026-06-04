package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteComment_RemovesReplySubtreeAndTopLevelCount(t *testing.T) {
	repo, db := newCommentRepoForTest(t)
	ctx := context.Background()

	fragmentID := "fragment-root-delete"
	seedCommentFixtures(t, db, fragmentID)

	root := &domain.Comment{
		UserID:     "u-commenter",
		Content:    "root",
		TargetType: "fragment",
		TargetID:   fragmentID,
	}
	if err := repo.CreateComment(ctx, root); err != nil {
		t.Fatalf("create root: %v", err)
	}

	reply := &domain.Comment{
		UserID:     "u-commenter",
		Content:    "reply",
		TargetType: "fragment",
		TargetID:   fragmentID,
		ParentID:   root.ID,
	}
	if err := repo.CreateComment(ctx, reply); err != nil {
		t.Fatalf("create reply: %v", err)
	}

	childReply := &domain.Comment{
		UserID:     "u-commenter",
		Content:    "reply-2",
		TargetType: "fragment",
		TargetID:   fragmentID,
		ParentID:   reply.ID,
	}
	if err := repo.CreateComment(ctx, childReply); err != nil {
		t.Fatalf("create nested reply: %v", err)
	}

	waitForFragmentComments(t, db, fragmentID, 1)

	if err := repo.DeleteComment(ctx, root.ID); err != nil {
		t.Fatalf("delete root: %v", err)
	}

	waitForFragmentComments(t, db, fragmentID, 0)

	var aliveCount int64
	if err := db.Model(&Comment{}).
		Where("target_type = ? AND target_id = ? AND deleted_at IS NULL", "fragment", fragmentID).
		Count(&aliveCount).Error; err != nil {
		t.Fatalf("count alive comments: %v", err)
	}
	if aliveCount != 0 {
		t.Fatalf("expected subtree deleted, got alive=%d", aliveCount)
	}
}

func TestDeleteReply_DoesNotChangeTopLevelCounter(t *testing.T) {
	repo, db := newCommentRepoForTest(t)
	ctx := context.Background()

	fragmentID := "fragment-reply-delete"
	seedCommentFixtures(t, db, fragmentID)

	root := &domain.Comment{
		UserID:     "u-commenter",
		Content:    "root",
		TargetType: "fragment",
		TargetID:   fragmentID,
	}
	if err := repo.CreateComment(ctx, root); err != nil {
		t.Fatalf("create root: %v", err)
	}

	reply := &domain.Comment{
		UserID:     "u-commenter",
		Content:    "reply",
		TargetType: "fragment",
		TargetID:   fragmentID,
		ParentID:   root.ID,
	}
	if err := repo.CreateComment(ctx, reply); err != nil {
		t.Fatalf("create reply: %v", err)
	}

	waitForFragmentComments(t, db, fragmentID, 1)

	if err := repo.DeleteComment(ctx, reply.ID); err != nil {
		t.Fatalf("delete reply: %v", err)
	}

	waitForFragmentComments(t, db, fragmentID, 1)

	var parent Comment
	if err := db.Where("id = ?", root.ID).First(&parent).Error; err != nil {
		t.Fatalf("query parent: %v", err)
	}
	if parent.ReplyCount != 0 {
		t.Fatalf("expected parent reply_count reset to 0, got %d", parent.ReplyCount)
	}
}

func newCommentRepoForTest(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS fragments (
		id TEXT PRIMARY KEY,
		comments INTEGER NOT NULL DEFAULT 0
	)`).Error; err != nil {
		t.Fatalf("create fragments table: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS comments (
		id TEXT PRIMARY KEY,
		author_id TEXT NOT NULL,
		content TEXT NOT NULL,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		parent_id TEXT NULL,
		root_id TEXT NULL,
		likes INTEGER NOT NULL DEFAULT 0,
		dislikes INTEGER NOT NULL DEFAULT 0,
		reply_count INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatalf("create comments table: %v", err)
	}
	repo := NewRepository(db, zap.NewNop(), config.RecommendationConfig{})
	return repo, db
}

func seedCommentFixtures(t *testing.T, db *gorm.DB, fragmentID string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO fragments (id, comments) VALUES (?, 0)`, fragmentID).Error; err != nil {
		t.Fatalf("seed fragment: %v", err)
	}
}

func waitForFragmentComments(t *testing.T, db *gorm.DB, fragmentID string, expected int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		var row struct {
			Comments int
		}
		if err := db.Table("fragments").Select("comments").Where("id = ?", fragmentID).Take(&row).Error; err != nil {
			t.Fatalf("query fragment: %v", err)
		}
		if row.Comments == expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("fragment comments mismatch, expected=%d actual=%d", expected, row.Comments)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
