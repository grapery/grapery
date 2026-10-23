package mysql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/config"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStoryboardRequestHistoryFindsOlderKeyAndScopesBoardAndOwner(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	// Production's user_id collation is MySQL-specific. Preserve all scalar
	// columns while using SQLite text for this isolated repository query test.
	statement := &gorm.Statement{DB: db}
	if err = statement.Parse(&StoryboardGenerationRun{}); err != nil {
		t.Fatal(err)
	}
	columns := []string{}
	for _, field := range statement.Schema.Fields {
		if field.DBName != "" {
			column := "`" + field.DBName + "` " + db.Dialector.DataTypeOf(field)
			if field.DBName == "user_id" {
				column = "`user_id` text"
			}
			if field.PrimaryKey {
				column += " PRIMARY KEY"
			}
			columns = append(columns, column)
		}
	}
	if err = db.Exec("CREATE TABLE storyboard_generation_runs (" + strings.Join(columns, ",") + ")").Error; err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db, zap.NewNop(), config.RecommendationConfig{})
	key := `turn-'"-A`
	for i, row := range []struct{ id, board, user, key string }{
		{"run-A", "board", "owner", key}, {"run-B", "board", "owner", "turn-B"}, {"other-owner", "board", "other", key}, {"other-board", "other", "owner", key},
	} {
		raw, _ := json.Marshal(map[string]any{"clientRequestId": row.key})
		model := StoryboardGenerationRun{ID: row.id, StoryboardID: row.board, UserID: row.user, StoryID: "story", RequestJSON: string(raw), CreatedAt: time.Unix(int64(100+i), 0)}
		if err = db.Create(&model).Error; err != nil {
			t.Fatal(err)
		}
	}
	found, err := repo.FindStoryboardGenerationRunsByClientRequest(context.Background(), "board", "owner", key)
	if err != nil || len(found) != 1 || found[0].ID != "run-A" {
		t.Fatalf("history lost/scoped incorrectly: found=%+v err=%v", found, err)
	}
	found, err = repo.FindStoryboardGenerationRunsByClientRequest(context.Background(), "board", "owner", "missing")
	if err != nil || len(found) != 0 {
		t.Fatalf("unexpected missing result %+v %v", found, err)
	}
	// Storage failures must propagate instead of pretending the key is new.
	if err = db.Migrator().DropTable(&StoryboardGenerationRun{}); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.FindStoryboardGenerationRunsByClientRequest(context.Background(), "board", "owner", key); err == nil {
		t.Fatal("missing table hidden as new request")
	}
}
