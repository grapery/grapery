package mysql

import (
	"context"
	"encoding/json"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func publicationPassedTrace() *domain.FragmentGenerationTrace {
	return &domain.FragmentGenerationTrace{BillingStatus: "settled", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}, ComicDocument: &domain.FragmentComicDocument{Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}, Pages: []domain.FragmentComicPageDocument{{BackgroundImageURL: "background", FlattenedImageURL: "accepted", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}}}}}
}
func TestPublicationManifestRejectsReplacementAndKeepsAcceptedFlattened(t *testing.T) {
	trace := publicationPassedTrace()
	for _, url := range []string{"background", "accepted"} {
		if err := ValidateFragmentMediaManifest(trace, []string{url}); err != nil {
			t.Fatal(err)
		}
	}
	for _, urls := range [][]string{{"replacement"}, {"accepted", "extra"}, nil} {
		if ValidateFragmentMediaManifest(trace, urls) == nil {
			t.Fatal("replacement bypassed audit")
		}
	}
}
func TestPublicationRequiresCurrentCompletedSettledTaskAndUnchangedContent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE fragment_generation_tasks(id TEXT PRIMARY KEY,user_id TEXT,status TEXT,request_json TEXT,result_json TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	trace := publicationPassedTrace()
	trace.PublicationManifest = &domain.FragmentPublicationManifest{Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}, Pages: []domain.FragmentPublicationPage{{PageIndex: 0, ImageURL: "accepted", Quality: &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}}}}
	metadata, _ := json.Marshal(trace)
	result, _ := json.Marshal(&domain.FragmentGenerationResult{GenerationTrace: trace})
	stored := &FragmentDB{ID: "draft", IsDraft: true, UserID: "owner", Content: "server current story", GenerationTaskID: "task", GenerationMetadata: string(metadata)}
	proposed := &domain.Fragment{BaseModel: common.BaseModel{ID: "draft"}, ExpectedGenerationTaskID: "task", GenerationTaskID: "task", Content: stored.Content, GenerationMetadata: stored.GenerationMetadata, MediaURLs: []string{"accepted"}}
	if err := db.Exec(`INSERT INTO fragment_generation_tasks VALUES(?,?,?,?,?)`, "task", "owner", "completed", `{"billingVersion":1}`, string(result)).Error; err != nil {
		t.Fatal(err)
	}
	guard := func() error {
		return db.Transaction(func(tx *gorm.DB) error {
			return ValidateFragmentPublicationAuthority(context.Background(), tx, stored, proposed)
		})
	}
	if err := guard(); err != nil {
		t.Fatal(err)
	}
	proposed.MediaURLs = []string{"unreviewed replacement"}
	if guard() == nil {
		t.Fatal("replacement passed")
	}
	proposed.MediaURLs = []string{"accepted"}
	proposed.Content = "stale client text"
	if guard() == nil {
		t.Fatal("stale text passed")
	}
	proposed.Content = stored.Content
	proposed.ExpectedGenerationTaskID = "old-task"
	if guard() == nil {
		t.Fatal("stale task passed")
	}
	proposed.ExpectedGenerationTaskID = "task"
	db.Exec(`UPDATE fragment_generation_tasks SET status='processing'`)
	stored.GenerationMetadata = ""
	proposed.GenerationMetadata = ""
	if guard() == nil {
		t.Fatal("generation in progress with empty metadata treated as legacy")
	}
	db.Exec(`UPDATE fragment_generation_tasks SET status='completed',result_json='{"generationTrace":{"billingStatus":"pending"}}'`)
	if guard() == nil {
		t.Fatal("pending bill published")
	}
	db.Exec(`UPDATE fragment_generation_tasks SET result_json=?`, string(result))
	if guard() == nil {
		t.Fatal("new task with missing quality treated as legacy")
	}
}

func TestLegacyPanelPublicationUsesPanelTaskAuthority(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE fragment_panel_generation_tasks(id TEXT PRIMARY KEY,user_id TEXT,status TEXT)`)
	db.Exec(`INSERT INTO fragment_panel_generation_tasks VALUES('panel','owner','completed')`)
	stored := &FragmentDB{ID: "draft", UserID: "owner", IsDraft: true, SourceType: string(domain.FragmentSourcePanelGeneration), SourceID: "panel"}
	proposed := &domain.Fragment{Content: "legacy"}
	if err := ValidateFragmentPublicationAuthority(context.Background(), db, stored, proposed); err != nil {
		t.Fatal(err)
	}
	db.Exec(`UPDATE fragment_panel_generation_tasks SET status='processing'`)
	if err := ValidateFragmentPublicationAuthority(context.Background(), db, stored, proposed); err == nil {
		t.Fatal("unfinished panel task published")
	}
}

func TestPublicationExactSelectedManifestAllowsDeletionWithoutApprovingExcludedPage(t *testing.T) {
	passed := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityPassed}
	blocked := &domain.FragmentGenerationQuality{Status: domain.FragmentQualityBlocked}
	trace := &domain.FragmentGenerationTrace{BillingStatus: "settled", Quality: blocked, ComicDocument: &domain.FragmentComicDocument{Quality: blocked, Pages: []domain.FragmentComicPageDocument{{PageIndex: 0, BackgroundImageURL: "a", Quality: passed}, {PageIndex: 1, BackgroundImageURL: "b", Quality: blocked}, {PageIndex: 2, BackgroundImageURL: "c", Quality: passed}}}, PublicationManifest: &domain.FragmentPublicationManifest{Quality: passed, Pages: []domain.FragmentPublicationPage{{PageIndex: 0, ImageURL: "flat-a", Quality: passed}, {PageIndex: 2, ImageURL: "flat-c", Quality: passed}}}}
	metadata, _ := json.Marshal(trace)
	if err := domain.ValidateFragmentPublicationQuality(string(metadata)); err != nil {
		t.Fatal("excluded page blocked subset", err)
	}
	if err := ValidateFragmentMediaManifest(trace, []string{"flat-a", "flat-c"}); err != nil {
		t.Fatal(err)
	}
	for _, urls := range [][]string{{"flat-a", "b", "flat-c"}, {"flat-c", "flat-a"}, {"replacement", "flat-c"}, {"a", "c"}} {
		if ValidateFragmentMediaManifest(trace, urls) == nil {
			t.Fatalf("bypass by %v", urls)
		}
	}
	trace.PublicationManifest.Quality = &domain.FragmentGenerationQuality{Status: domain.FragmentQualityUnreviewed}
	metadata, _ = json.Marshal(trace)
	if domain.ValidateFragmentPublicationQuality(string(metadata)) == nil {
		t.Fatal("unknown audit approved")
	}
	if trace.ComicDocument.Pages[1].Quality.Status != domain.FragmentQualityBlocked {
		t.Fatal("excluded failure erased")
	}
}
