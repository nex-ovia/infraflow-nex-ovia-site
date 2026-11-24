package main

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/gobold"
)

func TestCardSizes(t *testing.T) {
	for _, size := range []struct{ X, Y int }{{CardOG.X, CardOG.Y}, {CardIG.X, CardIG.Y}} {
		b, err := Card(CardOG, "A fairly long title that has to wrap across more than one line", "nex-ovia.com/blog", "nex·ovia")
		if size.X == CardIG.X {
			b, err = Card(CardIG, "Short", "", "nex·ovia")
		}
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Size(); got.X != size.X || got.Y != size.Y {
			t.Errorf("size = %v, want %dx%d", got, size.X, size.Y)
		}
	}
}

func TestWrapLimitsLinesAndWidth(t *testing.T) {
	f, err := face(gobold.TTF, 60)
	if err != nil {
		t.Fatal(err)
	}
	lines := wrap(f, strings.Repeat("word ", 60), 600, 3)
	if len(lines) != 3 || !strings.HasSuffix(lines[2], "…") {
		t.Fatalf("want 3 lines ending with …, got %q", lines)
	}
}
