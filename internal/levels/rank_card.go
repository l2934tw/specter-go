package levels

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"time"

	"github.com/fogleman/gg"
)

// RankCardData holds everything needed to render a rank card.
type RankCardData struct {
	Username  string
	Discrim   string
	AvatarURL string
	Level     int
	Rank      int
	XP        int64
	TotalMsgs int64
}

// RenderRankCard renders a clean, modern rank card. It intentionally avoids
// glow, blur and other luminous effects so it stays crisp at Discord sizes.
func RenderRankCard(ctx context.Context, d RankCardData) ([]byte, error) {
	const w, h = 1000, 320
	dc := gg.NewContext(w, h)

	// Background and card.
	dc.SetRGB(0.045, 0.050, 0.070)
	dc.Clear()
	dc.SetRGB(0.075, 0.085, 0.115)
	dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26)
	dc.Fill()

	// Thin accent rule.
	dc.SetRGB(0.38, 0.42, 0.98)
	dc.DrawRoundedRectangle(48, 44, 96, 5, 2.5)
	dc.Fill()

	const ax, ay, ar = 118.0, 160.0, 68.0
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

	// Status indicator.
	dc.SetRGB(0.075, 0.085, 0.115)
	dc.DrawCircle(ax+48, ay+48, 15)
	dc.Fill()
	dc.SetRGB(0.25, 0.90, 0.55)
	dc.DrawCircle(ax+48, ay+48, 9)
	dc.Fill()

	font := findFont()
	name := d.Username
	if d.Discrim != "" && d.Discrim != "0" {
		name = fmt.Sprintf("%s#%s", d.Username, d.Discrim)
	}
	if err := dc.LoadFontFace(font, 38); err == nil {
		dc.SetRGB(0.96, 0.97, 1.0)
		dc.DrawString(truncate(name, 28), 215, 102)
	}

	dc.SetRGBA(1, 1, 1, 0.05)
	dc.DrawRectangle(215, 116, 680, 1)
	dc.Fill()

	// Level badge.
	dc.SetRGBA(0.38, 0.42, 0.98, 0.14)
	dc.DrawRoundedRectangle(215, 133, 112, 38, 12)
	dc.Fill()
	if err := dc.LoadFontFace(font, 18); err == nil {
		dc.SetRGB(0.40, 0.44, 1.0)
		dc.DrawString(fmt.Sprintf("LEVEL %d", d.Level), 229, 158)
	}

	// Rank badge.
	dc.SetRGBA(1, 1, 1, 0.05)
	dc.DrawRoundedRectangle(337, 133, 112, 38, 12)
	dc.Fill()
	if err := dc.LoadFontFace(font, 18); err == nil {
		dc.SetRGB(0.90, 0.91, 0.95)
		dc.DrawString(fmt.Sprintf("#%d RANK", d.Rank), 351, 158)
	}

	if err := dc.LoadFontFace(font, 18); err == nil {
		dc.SetRGB(0.58, 0.61, 0.70)
		dc.DrawString(fmt.Sprintf("%d messages", d.TotalMsgs), 470, 158)
	}

	curBase := CalculateXPForLevel(d.Level)
	nextBase := CalculateXPForLevel(d.Level + 1)
	span := nextBase - curBase
	progress := 0.0
	if span > 0 {
		progress = float64(d.XP-curBase) / float64(span)
	}
	if progress < 0 { progress = 0 }
	if progress > 1 { progress = 1 }
	currentXP := d.XP - curBase
	if currentXP < 0 { currentXP = 0 }

	if err := dc.LoadFontFace(font, 17); err == nil {
		dc.SetRGB(0.58, 0.61, 0.70)
		dc.DrawString("EXPERIENCE", 215, 201)
		xpText := fmt.Sprintf("%d / %d XP", currentXP, span)
		tw, _ := dc.MeasureString(xpText)
		dc.SetRGB(0.92, 0.93, 0.97)
		dc.DrawString(xpText, 895-tw, 201)
	}

	const barX, barY, barW, barH = 215.0, 215.0, 680.0, 18.0
	dc.SetRGB(0.13, 0.145, 0.19)
	dc.DrawRoundedRectangle(barX, barY, barW, barH, 9)
	dc.Fill()
	if progress > 0 {
		dc.SetRGB(0.38, 0.42, 0.98)
		dc.DrawRoundedRectangle(barX, barY, barW*progress, barH, 9)
		dc.Fill()
	}

	if err := dc.LoadFontFace(font, 16); err == nil {
		dc.SetRGB(0.58, 0.61, 0.70)
		remaining := span - currentXP
		if remaining < 0 { remaining = 0 }
		dc.DrawString(fmt.Sprintf("%d XP remaining", remaining), barX, 263)
		pct := fmt.Sprintf("%.0f%%", progress*100)
		tw, _ := dc.MeasureString(pct)
		dc.SetRGB(0.92, 0.93, 0.97)
		dc.DrawString(pct, barX+barW-tw, 263)
	}

	return encodePNG(dc)
}

func fetchAvatar(ctx context.Context, url string) image.Image {
	if url == "" { return nil }
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil { return nil }
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return nil }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return nil }
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil { return nil }
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil { return nil }
	return img
}

func resizeToSquare(src image.Image, size int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < size; y++ {
		sy := b.Min.Y + y*sh/size
		for x := 0; x < size; x++ {
			sx := b.Min.X + x*sw/size
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}
