package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

type storyboardPageRepository struct {
	domain.Repository
	window        []*domain.Storyboard
	stories       map[string]*domain.Story
	publicEnabled bool
	calls         int
	limit, offset int
	err           error
}

func (r *storyboardPageRepository) StoryboardsByCreator(_ context.Context, _ string, limit, offset int) ([]*domain.Storyboard, error) {
	r.calls++
	r.limit, r.offset = limit, offset
	return r.window, r.err
}

func (r *storyboardPageRepository) UserSettings(context.Context, string) (*domain.UserSettings, error) {
	return &domain.UserSettings{ShowPublicStories: r.publicEnabled}, nil
}

func (r *storyboardPageRepository) IsFollowing(context.Context, string, string) (bool, error) {
	return false, nil
}

func (r *storyboardPageRepository) StoryByID(_ context.Context, id string) (*domain.Story, error) {
	story, ok := r.stories[id]
	if !ok {
		return nil, errors.New("story missing")
	}
	return story, nil
}

type storyboardPageCache struct {
	cache.Cache
	values map[string][]byte
}

func (c *storyboardPageCache) Get(_ context.Context, key string, dest interface{}) error {
	data, ok := c.values[key]
	if !ok {
		return errors.New("cache miss")
	}
	return json.Unmarshal(data, dest)
}

func (c *storyboardPageCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.values[key] = data
	return nil
}

func TestGetUserStoryboardPageVisibilityAndPagination(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name       string
		visibility []string
		wantIDs    []string
		wantMore   bool
	}{
		{name: "full hidden page still advances", visibility: []string{"private", "private"}, wantMore: true},
		{name: "mixed full page exposes only public", visibility: []string{"private", "public"}, wantIDs: []string{"board-1"}, wantMore: true},
		{name: "visible last partial page", visibility: []string{"public"}, wantIDs: []string{"board-0"}},
		{name: "hidden last partial page", visibility: []string{"private"}},
		{name: "empty end page"},
	}
	for _, tc := range cases {
		for _, cacheMode := range []string{"disabled", "miss", "hit"} {
			t.Run(tc.name+"/cache_"+cacheMode, func(t *testing.T) {
				repo := &storyboardPageRepository{publicEnabled: true, stories: map[string]*domain.Story{}}
				for i, visibility := range tc.visibility {
					id := fmt.Sprintf("story-%d", i)
					repo.window = append(repo.window, &domain.Storyboard{BaseModel: common.BaseModel{ID: fmt.Sprintf("board-%d", i)}, StoryID: id})
					repo.stories[id] = &domain.Story{Visibility: visibility}
				}
				svc := &Service{repo: repo, logger: zap.NewNop()}
				if cacheMode != "disabled" {
					c := &storyboardPageCache{values: map[string][]byte{}}
					if cacheMode == "hit" {
						if err := c.Set(ctx, cache.UserStoryboardsListKey("owner", 2, 4), repo.window, time.Minute); err != nil {
							t.Fatal(err)
						}
					}
					svc.cache = c
				}
				page, err := svc.GetUserStoryboardPage(ctx, "owner", "visitor", 2, 4)
				if err != nil {
					t.Fatal(err)
				}
				if page.HasMore != tc.wantMore || len(page.Storyboards) != len(tc.wantIDs) {
					t.Fatalf("page = (%d visible, hasMore %v), want (%d, %v)", len(page.Storyboards), page.HasMore, len(tc.wantIDs), tc.wantMore)
				}
				for i, id := range tc.wantIDs {
					if page.Storyboards[i].ID != id {
						t.Fatalf("visible board %d = %q, want %q", i, page.Storyboards[i].ID, id)
					}
				}
				wantCalls := 1
				if cacheMode == "hit" {
					wantCalls = 0
				}
				if repo.calls != wantCalls {
					t.Fatalf("repository calls = %d, want %d", repo.calls, wantCalls)
				}
				// A cached raw window must not be replaced by a viewer-filtered list.
				ownerPage, err := svc.GetUserStoryboardPage(ctx, "owner", "owner", 2, 4)
				if err != nil || len(ownerPage.Storyboards) != len(repo.window) {
					t.Fatalf("owner page lost raw items: page=%+v, err=%v", ownerPage, err)
				}
			})
		}
	}
}

func TestGetUserStoryboardPageNormalizesPagination(t *testing.T) {
	for _, tc := range []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, -10, 20, 0}, {-1, 7, 20, 7}, {200, -1, 100, 0}, {5, 10, 5, 10},
	} {
		t.Run(fmt.Sprintf("limit_%d_offset_%d", tc.limit, tc.offset), func(t *testing.T) {
			repo := &storyboardPageRepository{}
			svc := &Service{repo: repo, logger: zap.NewNop()}
			if _, err := svc.GetUserStoryboardPage(context.Background(), "owner", "owner", tc.limit, tc.offset); err != nil {
				t.Fatal(err)
			}
			if repo.limit != tc.wantLimit || repo.offset != tc.wantOffset {
				t.Fatalf("repository pagination = (%d, %d), want (%d, %d)", repo.limit, repo.offset, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}

func TestGetUserStoryboardsCompatibilityAndRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &storyboardPageRepository{window: []*domain.Storyboard{{StoryID: "private"}}}
	svc := &Service{repo: repo, logger: zap.NewNop()}
	boards, err := svc.GetUserStoryboards(ctx, "owner", "owner", 2, 0)
	if err != nil || len(boards) != 1 {
		t.Fatalf("legacy list API = %v, %v", boards, err)
	}
	repo.err = errors.New("repository unavailable")
	if page, err := svc.GetUserStoryboardPage(ctx, "owner", "owner", 2, 0); page != nil || !errors.Is(err, repo.err) {
		t.Fatalf("repository error = %v, %v", page, err)
	}
}
