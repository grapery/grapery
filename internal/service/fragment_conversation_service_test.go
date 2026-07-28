package service

import "testing"

func TestFragmentGenerationConversationClientMessageID(t *testing.T) {
	t.Run("uses client message id shared with the app", func(t *testing.T) {
		got := fragmentGenerationConversationClientMessageID("task-1", " client-message-1 ")
		if got != "client-message-1" {
			t.Fatalf("got %q, want client-message-1", got)
		}
	})

	t.Run("falls back for legacy callers", func(t *testing.T) {
		got := fragmentGenerationConversationClientMessageID("task-1", "")
		if got != "task-1:user_input" {
			t.Fatalf("got %q, want task-1:user_input", got)
		}
	})
}
