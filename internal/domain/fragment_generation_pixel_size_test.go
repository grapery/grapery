package domain

import (
	"strconv"
	"strings"
	"testing"
)

func parseWxH(t *testing.T, s string) (w, h int) {
	t.Helper()
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		t.Fatalf("expected WxH, got %q", s)
	}
	var err error
	w, err = strconv.Atoi(parts[0])
	if err != nil {
		t.Fatal(err)
	}
	h, err = strconv.Atoi(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	return w, h
}

func TestFragmentImagePixelSizeForAspectRatio_meetsArkSeedreamFloor(t *testing.T) {
	const minPx = 3686400 // Huoshan / Seedream 5.0 组图报错阈值（见图）
	aspects := []string{
		FragmentAspect1x1,
		FragmentAspect16x9,
		FragmentAspect9x16,
		FragmentAspect3x4,
		FragmentAspect4x3,
		"",
		"unknown",
	}
	for _, ar := range aspects {
		sz := FragmentImagePixelSizeForAspectRatio(ar)
		w, h := parseWxH(t, sz)
		if px := w * h; px < minPx {
			t.Fatalf("aspect %q -> %s = %d px (< %d)", ar, sz, px, minPx)
		}
	}
}
