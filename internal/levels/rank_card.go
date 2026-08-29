```go
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

// RenderRankCard renders a modern, minimal Discord-style rank card.
func RenderRankCard(ctx context.Context, d RankCardData) ([]byte, error) {
	const (
		w = 1000
		h = 320
	)

	dc := gg.NewContext(w, h)

	// ------------------------------------------------------------
	// Colors
	// ------------------------------------------------------------

	bg := struct {
		r, g, b float64
	}{
		0.055,
		0.063,
		0.090,
	}

	card := struct {
		r, g, b float64
	}{
		0.075,
		0.086,
		0.120,
	}

	cardLight := struct {
		r, g, b float64
	}{
		0.095,
		0.108,
		0.145,
	}

	primary := struct {
		r, g, b float64
	}{
		0.40,
		0.43,
		1.00,
	}

	primarySoft := struct {
		r, g, b float64
	}{
		0.29,
		0.32,
		0.82,
	}

	white := struct {
		r, g, b float64
	}{
		0.96,
		0.97,
		1.00,
	}

	muted := struct {
		r, g, b float64
	}{
		0.58,
		0.61,
		0.70,
	}

	darkBar := struct {
		r, g, b float64
	}{
		0.13,
		0.145,
		0.19,
	}

	// ------------------------------------------------------------
	// Background
	// ------------------------------------------------------------

	dc.SetRGB(bg.r, bg.g, bg.b)
	dc.Clear()

	// Subtle background glow.
	dc.SetRGBA(primary.r, primary.g, primary.b, 0.035)
	dc.DrawCircle(865, 50, 210)
	dc.Fill()

	dc.SetRGBA(primary.r, primary.g, primary.b, 0.025)
	dc.DrawCircle(80, 300, 180)
	dc.Fill()

	// ------------------------------------------------------------
	// Main Card
	// ------------------------------------------------------------

	dc.SetRGB(card.r, card.g, card.b)
	dc.DrawRoundedRectangle(18, 18, w-36, h-36, 28)
	dc.Fill()

	// Very subtle inner panel.
	dc.SetRGB(cardLight.r, cardLight.g, cardLight.b)
	dc.DrawRoundedRectangle(28, 28, w-56, h-56, 22)
	dc.SetRGBA(card.r, card.g, card.b, 0.42)
	dc.Fill()

	// Top accent line.
	dc.SetRGB(primary.r, primary.g, primary.b)
	dc.DrawRoundedRectangle(48, 44, 110, 5, 2.5)
	dc.Fill()

	// ------------------------------------------------------------
	// Avatar
	// ------------------------------------------------------------

	const (
		ax = 118.0
		ay = 160.0
		ar = 68.0
	)

	// Avatar glow.
	dc.SetRGBA(primary.r, primary.g, primary.b, 0.10)
	dc.DrawCircle(ax, ay, ar+14)
	dc.Fill()

	// Outer ring.
	dc.SetRGB(primarySoft.r, primarySoft.g, primarySoft.b)
	dc.DrawCircle(ax, ay, ar+7)
	dc.Fill()

	// Inner dark ring.
	dc.SetRGB(card.r, card.g, card.b)
	dc.DrawCircle(ax, ay, ar+3)
	dc.Fill()

	// Avatar image.
	if img := fetchAvatar(ctx, d.AvatarURL); img != nil {
		dc.Push()

		dc.DrawCircle(ax, ay, ar)
		dc.Clip()

		scaled := resizeToSquare(img, int(ar*2))
		dc.DrawImage(
			scaled,
			int(ax-ar),
			int(ay-ar),
		)

		dc.ResetClip()
		dc.Pop()
	} else {
		// Fallback avatar.
		dc.SetRGB(primarySoft.r, primarySoft.g, primarySoft.b)
		dc.DrawCircle(ax, ay, ar)
		dc.Fill()
	}

	// ------------------------------------------------------------
	// Online/status indicator
	// ------------------------------------------------------------

	dc.SetRGB(card.r, card.g, card.b)
	dc.DrawCircle(ax+48, ay+48, 15)
	dc.Fill()

	dc.SetRGB(0.25, 0.90, 0.55)
	dc.DrawCircle(ax+48, ay+48, 9)
	dc.Fill()

	// ------------------------------------------------------------
	// Username
	// ------------------------------------------------------------

	name := d.Username

	if d.Discrim != "" && d.Discrim != "0" {
		name = fmt.Sprintf("%s#%s", d.Username, d.Discrim)
	}

	if err := dc.LoadFontFace(findFont(), 38); err == nil {
		dc.SetRGB(white.r, white.g, white.b)
		dc.DrawString(name, 215, 102)
	}

	// Small separator under username.
	dc.SetRGBA(1, 1, 1, 0.045)
	dc.DrawRoundedRectangle(215, 116, 680, 1, 0.5)
	dc.Fill()

	// ------------------------------------------------------------
	// Level + Rank
	// ------------------------------------------------------------

	// Level badge.
	dc.SetRGBA(primary.r, primary.g, primary.b, 0.13)
	dc.DrawRoundedRectangle(215, 133, 112, 38, 12)
	dc.Fill()

	if err := dc.LoadFontFace(findFont(), 20); err == nil {
		dc.SetRGB(primary.r, primary.g, primary.b)
		dc.DrawString(fmt.Sprintf("LEVEL %d", d.Level), 230, 159)
	}

	// Rank badge.
	dc.SetRGBA(1, 1, 1, 0.045)
	dc.DrawRoundedRectangle(337, 133, 112, 38, 12)
	dc.Fill()

	if err := dc.LoadFontFace(findFont(), 20); err == nil {
		dc.SetRGB(white.r, white.g, white.b)
		dc.DrawString(fmt.Sprintf("#%d RANK", d.Rank), 352, 159)
	}

	// Messages.
	if err := dc.LoadFontFace(findFont(), 18); err == nil {
		dc.SetRGB(muted.r, muted.g, muted.b)

		messageText := fmt.Sprintf(
			"%d messages",
			d.TotalMsgs,
		)

		dc.DrawString(messageText, 470, 158)
	}

	// ------------------------------------------------------------
	// XP Calculation
	// ------------------------------------------------------------

	curBase := CalculateXPForLevel(d.Level)
	nextBase := CalculateXPForLevel(d.Level + 1)

	span := nextBase - curBase

	progress := 0.0

	if span > 0 {
		progress = float64(d.XP-curBase) / float64(span)
	}

	if progress < 0 {
		progress = 0
	}

	if progress > 1 {
		progress = 1
	}

	currentXP := d.XP - curBase

	if currentXP < 0 {
		currentXP = 0
	}

	// ------------------------------------------------------------
	// XP Header
	// ------------------------------------------------------------

	if err := dc.LoadFontFace(findFont(), 18); err == nil {
		dc.SetRGB(muted.r, muted.g, muted.b)
		dc.DrawString("EXPERIENCE", 215, 201)

		dc.SetRGB(white.r, white.g, white.b)

		xpText := fmt.Sprintf(
			"%d / %d XP",
			currentXP,
			span,
		)

		// Right aligned XP text.
		textWidth, _ := dc.MeasureString(xpText)

		dc.DrawString(
			xpText,
			895-textWidth,
			201,
		)
	}

	// ------------------------------------------------------------
	// Progress Bar
	// ------------------------------------------------------------

	const (
		barX = 215.0
		barY = 215.0
		barW = 680.0
		barH = 18.0
		radius = 9.0
	)

	// Bar shadow / glow.
	if progress > 0 {
		dc.SetRGBA(primary.r, primary.g, primary.b, 0.10)
		dc.DrawRoundedRectangle(
			barX-2,
			barY-2,
			barW*progress+4,
			barH+4,
			radius+2,
		)
		dc.Fill()
	}

	// Background.
	dc.SetRGB(darkBar.r, darkBar.g, darkBar.b)
	dc.DrawRoundedRectangle(
		barX,
		barY,
		barW,
		barH,
		radius,
	)
	dc.Fill()

	// Progress.
	if progress > 0 {
		dc.SetRGB(primary.r, primary.g, primary.b)

		dc.DrawRoundedRectangle(
			barX,
			barY,
			barW*progress,
			barH,
			radius,
		)

		dc.Fill()

		// Small highlight.
		dc.SetRGBA(1, 1, 1, 0.16)
		dc.DrawRoundedRectangle(
			barX+3,
			barY+3,
			maxFloat(barW*progress-6, 0),
			4,
			2,
		)
		dc.Fill()
	}

	// ------------------------------------------------------------
	// Bottom Stats
	// ------------------------------------------------------------

	if err := dc.LoadFontFace(findFont(), 16); err == nil {
		dc.SetRGB(muted.r, muted.g, muted.b)

		remaining := span - currentXP
		if remaining < 0 {
			remaining = 0
		}

		leftText := fmt.Sprintf(
			"%d XP remaining",
			remaining,
		)

		dc.DrawString(leftText, barX, 263)

		percentage := fmt.Sprintf(
			"%.0f%%",
			progress*100,
		)

		textWidth, _ := dc.MeasureString(percentage)

		dc.SetRGB(white.r, white.g, white.b)
		dc.DrawString(
			percentage,
			barX+barW-textWidth,
			263,
		)
	}

	// ------------------------------------------------------------
	// Encode PNG
	// ------------------------------------------------------------

	var buf bytes.Buffer

	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf(
			"encode rank card: %w",
			err,
		)
	}

	return buf.Bytes(), nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func fetchAvatar(ctx context.Context, url string) image.Image {
	if url == "" {
		return nil
	}

	reqCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	data, err := io.ReadAll(
		io.LimitReader(resp.Body, 8<<20),
	)
	if err != nil {
		return nil
	}

	img, _, err := image.Decode(
		bytes.NewReader(data),
	)
	if err != nil {
		return nil
	}

	return img
}

func resizeToSquare(src image.Image, size int) image.Image {
	dst := image.NewRGBA(
		image.Rect(0, 0, size, size),
	)

	b := src.Bounds()

	sw := b.Dx()
	sh := b.Dy()

	for y := 0; y < size; y++ {
		sy := b.Min.Y + y*sh/size

		for x := 0; x < size; x++ {
			sx := b.Min.X + x*sw/size

			dst.Set(
				x,
				y,
				src.At(sx, sy),
			)
		}
	}

	return dst
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}
```
