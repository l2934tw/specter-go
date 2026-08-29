package levels

import (
	"bytes"
	"context"
	"fmt"
	"image"

	"github.com/fogleman/gg"
)

// WelcomeCardData contains the data used to render a welcome card.
type WelcomeCardData struct {
	Username    string
	AvatarURL   string
	ServerName  string
	MemberCount int
}

// LevelUpCardData contains the data used to render a level-up card.
type LevelUpCardData struct {
	Username  string
	AvatarURL string
	Level     int
	Rank      int
	XP        int64
}

// RenderWelcomeCard renders a clean welcome image without glow effects.
func RenderWelcomeCard(ctx context.Context, d WelcomeCardData) ([]byte, error) {
	const w, h = 1000, 300
	dc := gg.NewContext(w, h)

	dc.SetRGB(0.045, 0.050, 0.070)
	dc.Clear()
	dc.SetRGB(0.075, 0.085, 0.115)
	dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26)
	dc.Fill()

	dc.SetRGB(0.38, 0.42, 0.98)
	dc.DrawRoundedRectangle(42, 42, 8, 216, 4)
	dc.Fill()

	const ax, ay, ar = 150.0, 150.0, 70.0
	dc.SetRGB(0.13, 0.15, 0.20)
	dc.DrawCircle(ax, ay, ar+7)
	dc.Fill()
	if img := fetchAvatar(ctx, d.AvatarURL); img != nil {
		dc.Push()
		dc.DrawCircle(ax, ay, ar)
		dc.Clip()
		dc.DrawImage(resizeToSquare(img, int(ar*2)), int(ax-ar), int(ay-ar))
		dc.ResetClip()
		dc.Pop()
	} else {
		dc.SetRGB(0.38, 0.42, 0.98)
		dc.DrawCircle(ax, ay, ar)
		dc.Fill()
	}

	font := findFont()
	if err := dc.LoadFontFace(font, 18); err == nil {
		dc.SetRGB(0.58, 0.61, 0.70)
		dc.DrawString("WELCOME TO", 270, 105)
	}
	if err := dc.LoadFontFace(font, 38); err == nil {
		dc.SetRGB(0.96, 0.97, 1.0)
		dc.DrawString(truncate(d.ServerName, 34), 270, 145)
	}
	if err := dc.LoadFontFace(font, 22); err == nil {
		dc.SetRGB(0.78, 0.80, 0.87)
		dc.DrawString(fmt.Sprintf("Welcome, %s", truncate(d.Username, 28)), 270, 185)
	}
	if err := dc.LoadFontFace(font, 17); err == nil {
		dc.SetRGB(0.52, 0.55, 0.64)
		dc.DrawString(fmt.Sprintf("Member #%d", d.MemberCount), 270, 220)
	}

	return encodePNG(dc)
}

// RenderLevelUpCard renders a compact level-up announcement card.
func RenderLevelUpCard(ctx context.Context, d LevelUpCardData) ([]byte, error) {
	const w, h = 1000, 300
	dc := gg.NewContext(w, h)

	dc.SetRGB(0.045, 0.050, 0.070)
	dc.Clear()
	dc.SetRGB(0.075, 0.085, 0.115)
	dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26)
	dc.Fill()

	// Accent block, intentionally flat: no glow.
	dc.SetRGB(0.38, 0.42, 0.98)
	dc.DrawRoundedRectangle(42, 42, 180, 216, 18)
	dc.Fill()

	if err := dc.LoadFontFace(findFont(), 16); err == nil {
		dc.SetRGB(0.84, 0.86, 1.0)
		dc.DrawStringAnchored("LEVEL UP", 132, 92, 0.5, 0.5)
	}
	if err := dc.LoadFontFace(findFont(), 64); err == nil {
		dc.SetRGB(1, 1, 1)
		dc.DrawStringAnchored(fmt.Sprintf("%d", d.Level), 132, 154, 0.5, 0.5)
	}
	if err := dc.LoadFontFace(findFont(), 15); err == nil {
		dc.SetRGB(0.84, 0.86, 1.0)
		dc.DrawStringAnchored("NEW LEVEL", 132, 214, 0.5, 0.5)
	}

	const ax, ay, ar = 282.0, 150.0, 58.0
	dc.SetRGB(0.13, 0.15, 0.20)
	dc.DrawCircle(ax, ay, ar+6)
	dc.Fill()
	if img := fetchAvatar(ctx, d.AvatarURL); img != nil {
		dc.Push()
		dc.DrawCircle(ax, ay, ar)
		dc.Clip()
		dc.DrawImage(resizeToSquare(img, int(ar*2)), int(ax-ar), int(ay-ar))
		dc.ResetClip()
		dc.Pop()
	} else {
		dc.SetRGB(0.38, 0.42, 0.98)
		dc.DrawCircle(ax, ay, ar)
		dc.Fill()
	}

	if err := dc.LoadFontFace(findFont(), 30); err == nil {
		dc.SetRGB(0.96, 0.97, 1.0)
		dc.DrawString(truncate(d.Username, 24), 365, 112)
	}
	if err := dc.LoadFontFace(findFont(), 19); err == nil {
		dc.SetRGB(0.58, 0.61, 0.70)
		dc.DrawString("Congratulations! You reached a new level.", 365, 145)
	}

	stats := fmt.Sprintf("Rank #%d    •    %d XP", d.Rank, d.XP)
	if err := dc.LoadFontFace(findFont(), 18); err == nil {
		dc.SetRGB(0.78, 0.80, 0.87)
		dc.DrawString(stats, 365, 195)
	}

	return encodePNG(dc)
}

func encodePNG(dc *gg.Context) ([]byte, error) {
	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("encode card: %w", err)
	}
	return buf.Bytes(), nil
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

var _ image.Image
