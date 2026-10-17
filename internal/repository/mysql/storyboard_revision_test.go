package mysql

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func storyboardRevisionRepository(t *testing.T) (*Repository, *domain.Storyboard, *domain.StoryboardRevisionBaseline) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	// Production uses a MySQL-specific creator_id collation. Build the same
	// scalar columns using SQLite storage types for transaction failure tests.
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(&Storyboard{}); err != nil {
		t.Fatal(err)
	}
	columns := []string{}
	for _, field := range stmt.Schema.Fields {
		if field.DBName != "" {
			col := "`" + field.DBName + "` " + db.Dialector.DataTypeOf(field)
			if field.DBName == "creator_id" {
				col = "`creator_id` text"
			}
			if field.PrimaryKey {
				col += " PRIMARY KEY"
			}
			columns = append(columns, col)
		}
	}
	if err := db.Exec("CREATE TABLE storyboards (" + strings.Join(columns, ",") + ")").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().CreateTable(&StoryboardScene{}); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db, zap.NewNop(), config.RecommendationConfig{})
	if err = db.Create(&Storyboard{ID: "board", StoryID: "story", UserID: "owner", Title: "title", Content: "accepted", WorkflowStatus: domain.WorkflowStatusContentReady}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&StoryboardScene{ID: "old", StoryboardID: "board", Title: "accepted scene", Sequence: 0, Characters: "[]", ContextSnapshot: "{}", ReferenceKeys: "[]", VisualStateJSON: "{}", ComicTextsJSON: "[]"}).Error; err != nil {
		t.Fatal(err)
	}
	var model Storyboard
	if err = db.First(&model, "id = ?", "board").Error; err != nil {
		t.Fatal(err)
	}
	board := ModelToStoryboard(&model)
	var row StoryboardScene
	if err = db.First(&row, "id = ?", "old").Error; err != nil {
		t.Fatal(err)
	}
	scenes := []domain.StoryboardScene{*repo.storyboardSceneToDomain(row)}
	baseline := &domain.StoryboardRevisionBaseline{Revision: domain.StoryboardContentRevision(board, scenes), Content: board.Content, Scenes: scenes}
	return repo, board, baseline
}

func TestStoryboardRevisionAtomicRollbackPreservesOldResult(t *testing.T) {
	repo, board, baseline := storyboardRevisionRepository(t)
	if err := repo.db.Exec("CREATE TRIGGER reject_scene BEFORE INSERT ON storyboard_scenes WHEN NEW.title = 'reject' BEGIN SELECT RAISE(ABORT, 'simulated scene write failure'); END;").Error; err != nil {
		t.Fatal(err)
	}
	board.Content = "new"
	board.StoryboardScenes = []domain.StoryboardScene{{Title: "reject"}}
	if err := repo.CommitStoryboardRevision(context.Background(), board, baseline); err == nil {
		t.Fatal("expected injected insertion failure")
	}
	var persisted Storyboard
	repo.db.First(&persisted, "id = ?", "board")
	var count int64
	repo.db.Model(&StoryboardScene{}).Where("id = ?", "old").Count(&count)
	if persisted.Content != "accepted" || count != 1 {
		t.Fatalf("failed commit damaged accepted result: content=%q scenes=%d", persisted.Content, count)
	}
}

func TestStoryboardRevisionCASRejectsStaleAndPublished(t *testing.T) {
	repo, board, baseline := storyboardRevisionRepository(t)
	repo.db.Model(&Storyboard{}).Where("id = ?", "board").Update("content", "edited elsewhere")
	board.Content = "candidate"
	board.StoryboardScenes = []domain.StoryboardScene{{Title: "candidate"}}
	if err := repo.CommitStoryboardRevision(context.Background(), board, baseline); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("stale commit error = %v", err)
	}
	repo.db.Model(&Storyboard{}).Where("id = ?", "board").Update("workflow_status", domain.WorkflowStatusPublished)
	if err := repo.CommitStoryboardRevision(context.Background(), board, baseline); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("published commit error = %v", err)
	}
}

func TestStoryboardRevisionCommitKeepsUntargetedStableRows(t *testing.T) {
	repo, board, baseline := storyboardRevisionRepository(t)
	board.Content = "revised"
	board.StoryboardScenes = []domain.StoryboardScene{baseline.Scenes[0], {Title: "new scene", Sequence: 1}}
	if err := repo.CommitStoryboardRevision(context.Background(), board, baseline); err != nil {
		t.Fatal(err)
	}
	var count int64
	repo.db.Model(&StoryboardScene{}).Count(&count)
	if count != 2 || board.StoryboardScenes[0].ID != "old" || board.StoryboardScenes[1].ID == "" {
		t.Fatalf("unstable result: count=%d scenes=%+v", count, board.StoryboardScenes)
	}
}
