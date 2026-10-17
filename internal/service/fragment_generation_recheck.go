package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

type fragmentPageAuditResult struct {
	Issues   []domain.FragmentConsistencyIssue
	Tokens   int
	Provider string
	Audited  int
	Expected int
	Skipped  string
}

func recheckFragmentDocument(ctx context.Context, original *domain.FragmentComicDocument, audit func(domain.FragmentComicPageDocument) fragmentPageAuditResult) (*domain.FragmentComicDocument, []domain.FragmentConsistencyIssue, int, error) {
	raw, err := json.Marshal(original)
	if err != nil {
		return nil, nil, 0, err
	}
	var doc domain.FragmentComicDocument
	if err = json.Unmarshal(raw, &doc); err != nil {
		return nil, nil, 0, err
	}
	issues := []domain.FragmentConsistencyIssue{}
	tokens := 0
	for i := range doc.Pages {
		page := &doc.Pages[i]
		if page.Quality == nil || page.Quality.Status != domain.FragmentQualityUnreviewed {
			continue
		}
		if err = ctx.Err(); err != nil {
			return nil, nil, tokens, err
		}
		if strings.TrimSpace(page.BackgroundImageURL) == "" {
			return nil, nil, tokens, fmt.Errorf("page %d has no rendered artwork", page.PageIndex+1)
		}
		result := audit(*page)
		tokens += result.Tokens
		page.Quality = fragmentGenerationQuality(result.Issues, result.Skipped, result.Provider, result.Expected, result.Audited)
		for _, issue := range result.Issues {
			issue.SceneIndex = page.PageIndex
			issues = append(issues, issue)
		}
	}
	reconcileFragmentDocumentQuality(&doc)
	return &doc, issues, tokens, nil
}

// RecheckQuality retries only incomplete audits of the current owner's rendered
// draft. It never calls a renderer, starts a new task, or charges image fees.
// Retrying unavailable quality infrastructure is sponsored by the platform.
func (s *FragmentGenerationService) RecheckQuality(ctx context.Context, taskID, userID string) (*domain.FragmentGenerationTask, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil || task == nil {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorNotFound, "task not found", err)
	}
	if task.UserID != userID {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorForbidden, "task is not owned by this user", nil)
	}
	if task.Status != "completed" || task.Result == nil || task.Result.ComicDocument == nil {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "only rendered completed tasks can be rechecked", nil)
	}
	if task.Result.Quality != nil && task.Result.Quality.Status == domain.FragmentQualityPassed {
		return task, nil
	}
	draftID := task.Result.DraftFragmentID
	unlock := s.lockFragmentDraftGeneration(draftID)
	defer unlock()
	draft, err := s.fragmentRepo.GetByID(ctx, draftID)
	if err != nil || draft == nil || draft.UserID != userID || !draft.IsDraft || draft.GenerationTaskID != taskID {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "the draft changed; reopen its latest task", err)
	}
	trace := parseFragmentGenerationTraceMetadata(draft.GenerationMetadata)
	if trace == nil || trace.ComicDocument == nil {
		return nil, domain.NewFragmentGenerationError(domain.FragmentGenerationErrorConflict, "audit source document is unavailable", nil)
	}
	priorMetadata := draft.GenerationMetadata
	recheckedPages := map[int]bool{}
	audit := func(page domain.FragmentComicPageDocument) fragmentPageAuditResult {
		recheckedPages[page.PageIndex] = true
		scene := domain.FragmentScenePlan{Index: 0, SceneDesc: page.PageIntent, ComicPage: &page.Plan}
		baseline := []string{}
		for _, previous := range trace.ComicDocument.Pages {
			if previous.PageIndex == page.PageIndex-1 && previous.BackgroundImageURL != "" {
				baseline = append(baseline, previous.BackgroundImageURL)
			}
		}
		issues, tokens, provider, audited, skipped := s.checkFragmentConsistency(ctx, taskID, trace.VisualBible, trace.VisualEvidence, []domain.FragmentScenePlan{scene}, trace.ReferenceAssets, baseline, []string{page.BackgroundImageURL}, &domain.FragmentConsistencyPolicy{Level: "strong"})
		return fragmentPageAuditResult{issues, tokens, provider, audited, 1 + len(baseline), skipped}
	}
	doc, issues, tokens, err := recheckFragmentDocument(ctx, trace.ComicDocument, audit)
	if err != nil {
		return nil, err
	}
	if len(recheckedPages) == 0 {
		return task, nil
	}
	// Preserve diagnostics for pages that were not rechecked, including blocked pages.
	for _, issue := range trace.ConsistencyIssues {
		if !recheckedPages[issue.SceneIndex] {
			issues = append(issues, issue)
		}
	}
	trace.ComicDocument = doc
	trace.Quality = doc.Quality
	trace.ConsistencyIssues = issues
	trace.Metrics = append(trace.Metrics, domain.FragmentGenerationStepMetric{Name: "quality_recheck", Tokens: tokens, Note: "platform sponsored; no image generation"})
	task.Result.ComicDocument = doc
	task.Result.Quality = doc.Quality
	task.Result.GenerationTrace = trace
	task.Result.ConsistencyIssues = issues
	task.Result.TokensUsed += tokens
	if err = s.fragmentGenRepo.UpdateRecheckedQuality(ctx, taskID, draftID, userID, priorMetadata, marshalFragmentGenerationMetadata(trace), task.Result); err != nil {
		return nil, err
	}
	task.TokensUsed = task.Result.TokensUsed
	task.UpdatedAt = time.Now().Unix()
	return task, nil
}
