package genapi

import (
	"reflect"
	"testing"
)

func TestCollectImages_dedupePrimaryWithExtras(t *testing.T) {
	got := collectImages("https://example.com/a.png", []string{"https://example.com/a.png", "https://example.com/b.png"}, 0)
	want := []string{"https://example.com/a.png", "https://example.com/b.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectImages: got %v want %v", got, want)
	}
}

func TestCollectImages_primaryOnly(t *testing.T) {
	got := collectImages("https://example.com/one.jpg", nil, 0)
	want := []string{"https://example.com/one.jpg"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectImages: got %v want %v", got, want)
	}
}
