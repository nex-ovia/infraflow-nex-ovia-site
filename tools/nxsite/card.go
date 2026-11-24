package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Card sizes: Open Graph / LinkedIn / X, and Instagram portrait.
var (
	CardOG = image.Pt(1200, 630)
	CardIG = image.Pt(1080, 1350)
)

// Brand palette (theme/theme.toml).
var (
	cardBG     = color.RGBA{0x0a, 0x0a, 0x0f, 0xff}
	cardText   = color.RGBA{0xf0, 0xf0, 0xf5, 0xff}
	cardMuted  = color.RGBA{0x90, 0x90, 0xa8, 0xff}
	cardAccent = color.RGBA{0x6c, 0x63, 0xff, 0xff}
	cardTeal   = color.RGBA{0x00, 0xd4, 0xaa, 0xff}
)

// Card renders a share image: brand mark, wrapped title, subtitle line, accent bars.
func Card(size image.Point, title, subtitle, mark string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	draw.Draw(img, img.Bounds(), &image.Uniform{cardBG}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, size.X, 12), &image.Uniform{cardAccent}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 12, size.X/3, 18), &image.Uniform{cardTeal}, image.Point{}, draw.Src)

	pad := size.X / 15
	titleSize := float64(size.X) / 17
	bold, err := face(gobold.TTF, titleSize)
	if err != nil {
		return nil, err
	}
	reg, err := face(goregular.TTF, titleSize*0.45)
	if err != nil {
		return nil, err
	}
	markFace, err := face(gobold.TTF, titleSize*0.5)
	if err != nil {
		return nil, err
	}

	drawText(img, markFace, cardAccent, pad, pad+int(titleSize*0.6), mark)
	lines := wrap(bold, title, size.X-2*pad, 4)
	lineH := int(titleSize * 1.2)
	y := size.Y/2 - (len(lines)*lineH)/2 + lineH/2
	for _, l := range lines {
		drawText(img, bold, cardText, pad, y, l)
		y += lineH
	}
	if subtitle != "" {
		drawText(img, reg, cardMuted, pad, size.Y-pad, subtitle)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func face(ttf []byte, size float64) (font.Face, error) {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
}

func drawText(img draw.Image, f font.Face, c color.Color, x, y int, s string) {
	d := &font.Drawer{Dst: img, Src: &image.Uniform{c}, Face: f, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

// wrap breaks s into at most maxLines lines no wider than width pixels; overflow ends with "…".
func wrap(f font.Face, s string, width, maxLines int) []string {
	var lines []string
	cur := ""
	for _, w := range strings.Fields(s) {
		try := strings.TrimSpace(cur + " " + w)
		if font.MeasureString(f, try).Ceil() <= width || cur == "" {
			cur = try
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		lines[maxLines-1] = strings.TrimRight(lines[maxLines-1], " .,;:") + "…"
	}
	return lines
}
