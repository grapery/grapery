package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

func characterMutationTestRepo(t *testing.T) *Repository {
	t.Helper()
	r := characterExecutionTestRepo(t)
	statement := &gorm.Statement{DB: r.db}
	if err := statement.Parse(&Character{}); err != nil {
		t.Fatal(err)
	}
	var columns []string
	for _, name := range statement.Schema.DBNames {
		field := statement.Schema.FieldsByDBName[name]
		definition := "`" + name + "` TEXT"
		if field.DataType == "time" {
			definition = "`" + name + "` DATETIME"
		}
		if name == "id" {
			definition += " PRIMARY KEY"
		}
		columns = append(columns, definition)
	}
	if err := r.db.Exec("CREATE TABLE characters (" + strings.Join(columns, ",") + ")").Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, deleted_at DATETIME)").Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.Exec("INSERT INTO users (id) VALUES ('owner')").Error; err != nil {
		t.Fatal(err)
	}
	if err := r.db.Create(&Character{ID: "character", StoryID: "story", UserID: "owner", Name: "雨", Description: "old", Portrait: "old-portrait", ViewsJSON: `{"sheet":"old-views"}`}).Error; err != nil {
		t.Fatal(err)
	}
	return r
}
func TestCharacterRetryBackfillUsesCurrentFieldsAndDoesNotOverwriteOwnerEdits(t *testing.T) {
	r := characterMutationTestRepo(t)
	ctx := context.Background()
	stale, err := r.CharacterByID(ctx, "character")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.db.Model(&Character{}).Where("id = ?", stale.ID).Updates(map[string]any{"description": "owner edit", "appearance": "owner appearance", "portrait": "new portrait", "views_json": `{"sheet":"new views"}`}).Error; err != nil {
		t.Fatal(err)
	}
	task := &domain.CharacterGenerationTask{ID: "task", UserID: "owner", StoryID: "story", Status: domain.CharacterGenerationStatusPending}
	if err := r.CreateCharacterGenerationTask(ctx, task); err != nil {
		t.Fatal(err)
	}
	lease, err := r.ClaimCharacterTask(ctx, task.ID, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.PatchCharacterFields(ctx, lease, stale, map[string]any{"description": "AI overwrite", "appearance": "AI overwrite", "background": "AI missing field"}, true); err != nil {
		t.Fatal(err)
	}
	if stale.Description != "owner edit" || stale.Appearance != "owner appearance" || stale.Background != "AI missing field" || stale.Portrait != "new portrait" || stale.Views.Sheet != "new views" {
		t.Fatalf("lost owner state: %+v", stale)
	}
	bad := *lease
	bad.UserID = "attacker"
	if err := r.PatchCharacterFields(ctx, &bad, stale, map[string]any{"description": "bad"}, true); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestCharacterExplicitPatchPreservesConcurrentAssetsAndAcceptsEmptyString(t *testing.T) {
	r := characterMutationTestRepo(t)
	ctx := context.Background()
	stale, err := r.CharacterByID(ctx, "character")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.db.Model(&Character{}).Where("id = ?", stale.ID).Updates(map[string]any{"portrait": "generated meanwhile", "avatar": "new-avatar", "views_json": `{"sheet":"new-views"}`}).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.PatchCharacterFields(ctx, nil, stale, map[string]any{"description": "", "name": "renamed"}, false); err != nil {
		t.Fatal(err)
	}
	if stale.Description != "" || stale.Name != "renamed" || stale.Portrait != "generated meanwhile" || stale.Avatar != "new-avatar" || stale.Views.Sheet != "new-views" {
		t.Fatalf("stale full row write: %+v", stale)
	}
	bad := *stale
	bad.UserID = "attacker"
	bad.Author = nil
	if err := r.PatchCharacterFields(ctx, nil, &bad, map[string]any{"avatar": "bad"}, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal(err)
	}
}
func TestCharacterConcurrentIndependentEditsBothSurvive(t *testing.T) {
	r := characterMutationTestRepo(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	for field, value := range map[string]string{"description": "owner description", "appearance": "owner appearance"} {
		snapshot, err := r.CharacterByID(ctx, "character")
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func(field, value string, ch *domain.Character) {
			defer wg.Done()
			if err := r.PatchCharacterFields(ctx, nil, ch, map[string]any{field: value}, false); err != nil {
				t.Error(err)
			}
		}(field, value, snapshot)
	}
	wg.Wait()
	ch, err := r.CharacterByID(ctx, "character")
	if err != nil || ch.Description != "owner description" || ch.Appearance != "owner appearance" {
		t.Fatal(ch, err)
	}
}
func TestCharacterOldVisualResponsesCannotReplaceNewAssetsOrSettings(t *testing.T) {
	for _, field := range []string{"portrait", "views_json", "appearance"} {
		t.Run(field, func(t *testing.T) {
			r := characterMutationTestRepo(t)
			ctx := context.Background()
			ch, err := r.CharacterByID(ctx, "character")
			if err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{"portrait": ch.Portrait, "views_sheet": ch.Views.Sheet, "inputs": domain.CharacterVisualInputSignature(ch)}
			value := "owner updated"
			if field == "views_json" {
				value = `{"sheet":"owner updated"}`
			}
			if err := r.db.Model(&Character{}).Where("id = ?", ch.ID).Update(field, value).Error; err != nil {
				t.Fatal(err)
			}
			if err := r.UpdateCharacterVisualAssetsIfUnchanged(ctx, nil, ch, map[string]any{"portrait": "late output", "portrait_generation_status": "generated"}, expected); !errors.Is(err, domain.ErrCharacterEditConflict) {
				t.Fatalf("old visual accepted %s: %v", field, err)
			}
			var saved Character
			if err := r.db.First(&saved, "id = ?", ch.ID).Error; err != nil {
				t.Fatal(err)
			}
			if saved.Portrait == "late output" {
				t.Fatal("old output saved")
			}
		})
	}
}
func TestCharacterGuardedVisualWriteStillFillsOnlyEmptyFallbackAvatar(t *testing.T) {
	r := characterMutationTestRepo(t)
	ctx := context.Background()
	ch, err := r.CharacterByID(ctx, "character")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"portrait": ch.Portrait, "views_sheet": ch.Views.Sheet, "inputs": domain.CharacterVisualInputSignature(ch)}
	if err := r.db.Model(&Character{}).Where("id = ?", ch.ID).Update("avatar", "owner avatar").Error; err != nil {
		t.Fatal(err)
	}
	if err := r.UpdateCharacterVisualAssetsIfUnchanged(ctx, nil, ch, map[string]any{"portrait": "new image", "avatar": "fallback", "poster": "fallback"}, expected); err != nil {
		t.Fatal(err)
	}
	if ch.Portrait != "new image" || ch.Avatar != "owner avatar" || ch.Poster != "fallback" {
		t.Fatal(fmt.Sprint(ch))
	}
}
