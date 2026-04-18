package service

import "testing"

func TestMergeNotificationMapsDeepMergePush(t *testing.T) {
	dst := map[string]interface{}{
		"push": map[string]interface{}{
			"enabled":     true,
			"newLike":     true,
			"newComment":  false,
		},
		"email": map[string]interface{}{"enabled": true},
	}
	src := map[string]interface{}{
		"push": map[string]interface{}{
			"enabled": false,
		},
	}
	mergeNotificationMaps(dst, src)
	push := dst["push"].(map[string]interface{})
	if push["enabled"] != false {
		t.Fatalf("expected push.enabled false, got %v", push["enabled"])
	}
	if push["newLike"] != true {
		t.Fatalf("expected newLike preserved, got %v", push["newLike"])
	}
	if push["newComment"] != false {
		t.Fatalf("expected newComment preserved, got %v", push["newComment"])
	}
}
