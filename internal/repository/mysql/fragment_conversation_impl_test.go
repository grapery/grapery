package mysql

import (
	"context"
	"strconv"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAppendFragmentConversationMessage_DedupesByClientMessageID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&FragmentConversationMessageDB{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := &Repository{db: db}
	ctx := context.Background()
	msg := &domain.FragmentConversationMessage{
		FragmentID:      "frag-1",
		UserID:          "user-1",
		Role:            domain.FragmentConversationRoleUser,
		MessageType:     domain.FragmentConversationTypeUserInput,
		Text:            "hello",
		ClientMessageID: "client-1",
	}
	if err := repo.AppendFragmentConversationMessage(ctx, msg); err != nil {
		t.Fatalf("append first: %v", err)
	}
	dup := *msg
	dup.ID = ""
	dup.Text = "hello again"
	if err := repo.AppendFragmentConversationMessage(ctx, &dup); err != nil {
		t.Fatalf("append duplicate: %v", err)
	}
	items, err := repo.ListFragmentConversationMessages(ctx, "frag-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
}

func TestAppendFragmentConversationMessage_PreservesClientCreatedAt(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&FragmentConversationMessageDB{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := &Repository{db: db}
	ctx := context.Background()
	const clientCreatedAt = int64(1234567890)
	msg := &domain.FragmentConversationMessage{
		FragmentID:      "frag-1",
		UserID:          "user-1",
		Role:            domain.FragmentConversationRoleUser,
		MessageType:     domain.FragmentConversationTypeUserInput,
		Text:            "hello",
		ClientMessageID: "client-1",
		CreatedAt:       clientCreatedAt,
	}
	if err := repo.AppendFragmentConversationMessage(ctx, msg); err != nil {
		t.Fatalf("append: %v", err)
	}
	items, err := repo.ListFragmentConversationMessages(ctx, "frag-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].CreatedAt != clientCreatedAt {
		t.Fatalf("expected createdAt %d, got %d", clientCreatedAt, items[0].CreatedAt)
	}
}

func TestListFragmentConversationMessagesPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&FragmentConversationMessageDB{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo := &Repository{db: db}
	ctx := context.Background()
	fragmentID := "frag-page"
	for i := 1; i <= 5; i++ {
		row := &FragmentConversationMessageDB{
			ID:              "msg-" + strconv.Itoa(i),
			FragmentID:      fragmentID,
			UserID:          "user-1",
			Role:            "user",
			MessageType:     "user_input",
			Text:            "hello " + strconv.Itoa(i),
			ClientMessageID: "client-" + strconv.Itoa(i),
			Sequence:        i,
			CreatedAt:       int64(i * 1000),
		}
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}

	latest, hasMore, err := repo.ListFragmentConversationMessagesPage(ctx, fragmentID, 2, 0)
	if err != nil {
		t.Fatalf("latest page: %v", err)
	}
	if !hasMore {
		t.Fatalf("expected hasMore on latest page")
	}
	if len(latest) != 2 || latest[0].Text != "hello 4" || latest[1].Text != "hello 5" {
		t.Fatalf("unexpected latest page: %+v", latest)
	}

	older, hasMoreOlder, err := repo.ListFragmentConversationMessagesPage(ctx, fragmentID, 2, latest[0].CreatedAt)
	if err != nil {
		t.Fatalf("older page: %v", err)
	}
	if !hasMoreOlder {
		t.Fatalf("expected hasMore on older page")
	}
	if len(older) != 2 || older[0].Text != "hello 2" || older[1].Text != "hello 3" {
		t.Fatalf("unexpected older page: %+v", older)
	}

	oldest, hasMoreOldest, err := repo.ListFragmentConversationMessagesPage(ctx, fragmentID, 2, older[0].CreatedAt)
	if err != nil {
		t.Fatalf("oldest page: %v", err)
	}
	if hasMoreOldest {
		t.Fatalf("expected no more pages")
	}
	if len(oldest) != 1 || oldest[0].Text != "hello 1" {
		t.Fatalf("unexpected oldest page: %+v", oldest)
	}
}
