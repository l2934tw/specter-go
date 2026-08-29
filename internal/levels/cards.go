package levels

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fogleman/gg"
)

type WelcomeCardData struct {
	Username      string
	AvatarURL     string
	ServerName    string
	MemberCount   int
	BackgroundURL string
}

type LevelUpCardData struct { Username string; AvatarURL string; Level int; Rank int; XP int64 }

func RenderWelcomeCard(ctx context.Context, d WelcomeCardData) ([]byte, error) {
	const w, h = 1080, 1080
	dc := gg.NewContext(w, h)
	if bg := fetchBackground(ctx, d.BackgroundURL); bg != nil {
		dc.DrawImageAnchored(coverImage(bg, w, h), w/2, h/2, 0.5, 0.5)
		dc.SetRGBA(0.02, 0.025, 0.04, 0.58)
		dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26); dc.Fill()
	} else {
		dc.SetRGB(0.045, 0.050, 0.070); dc.Clear(); dc.SetRGB(0.075, 0.085, 0.115)
		dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26); dc.Fill()
	}
	dc.SetRGB(0.38, 0.42, 0.98); dc.DrawRoundedRectangle(42, 42, 8, 216, 4); dc.Fill()
	const ax, ay, ar = 150.0, 150.0, 70.0
	dc.SetRGB(0.13, 0.15, 0.20); dc.DrawCircle(ax, ay, ar+7); dc.Fill()
	if img := fetchAvatar(ctx, d.AvatarURL); img != nil { dc.Push(); dc.DrawCircle(ax, ay, ar); dc.Clip(); dc.DrawImage(resizeToSquare(img, int(ar*2)), int(ax-ar), int(ay-ar)); dc.ResetClip(); dc.Pop() } else { dc.SetRGB(0.38, 0.42, 0.98); dc.DrawCircle(ax, ay, ar); dc.Fill() }
	font := findFont()
	if err := dc.LoadFontFace(font, 18); err == nil { dc.SetRGB(0.72, 0.75, 0.83); dc.DrawString("WELCOME TO", 270, 105) }
	if err := dc.LoadFontFace(font, 38); err == nil { dc.SetRGB(0.96, 0.97, 1.0); dc.DrawString(truncate(d.ServerName, 34), 270, 145) }
	if err := dc.LoadFontFace(font, 22); err == nil { dc.SetRGB(0.82, 0.84, 0.90); dc.DrawString(fmt.Sprintf("Welcome, %s", truncate(d.Username, 28)), 270, 185) }
	if err := dc.LoadFontFace(font, 17); err == nil { dc.SetRGB(0.64, 0.67, 0.75); dc.DrawString(fmt.Sprintf("Member #%d", d.MemberCount), 270, 220) }
	return encodePNG(dc)
}

func RenderLevelUpCard(ctx context.Context, d LevelUpCardData) ([]byte, error) {
	const w, h = 1080, 1080; dc := gg.NewContext(w, h)
	dc.SetRGB(0.045, 0.050, 0.070); dc.Clear(); dc.SetRGB(0.075, 0.085, 0.115); dc.DrawRoundedRectangle(18, 18, w-36, h-36, 26); dc.Fill()
	dc.SetRGB(0.38, 0.42, 0.98); dc.DrawRoundedRectangle(42, 42, 180, 216, 18); dc.Fill()
	if err := dc.LoadFontFace(findFont(), 16); err == nil { dc.SetRGB(0.84, 0.86, 1.0); dc.DrawStringAnchored("LEVEL UP", 132, 92, 0.5, 0.5) }
	if err := dc.LoadFontFace(findFont(), 64); err == nil { dc.SetRGB(1, 1, 1); dc.DrawStringAnchored(fmt.Sprintf("%d", d.Level), 132, 154, 0.5, 0.5) }
	if err := dc.LoadFontFace(findFont(), 15); err == nil { dc.SetRGB(0.84, 0.86, 1.0); dc.DrawStringAnchored("NEW LEVEL", 132, 214, 0.5, 0.5) }
	const ax, ay, ar = 282.0, 150.0, 58.0; dc.SetRGB(0.13, 0.15, 0.20); dc.DrawCircle(ax, ay, ar+6); dc.Fill()
	if img := fetchAvatar(ctx, d.AvatarURL); img != nil { dc.Push(); dc.DrawCircle(ax, ay, ar); dc.Clip(); dc.DrawImage(resizeToSquare(img, int(ar*2)), int(ax-ar), int(ay-ar)); dc.ResetClip(); dc.Pop() } else { dc.SetRGB(0.38, 0.42, 0.98); dc.DrawCircle(ax, ay, ar); dc.Fill() }
	if err := dc.LoadFontFace(findFont(), 30); err == nil { dc.SetRGB(0.96, 0.97, 1.0); dc.DrawString(truncate(d.Username, 24), 365, 112) }
	if err := dc.LoadFontFace(findFont(), 19); err == nil { dc.SetRGB(0.58, 0.61, 0.70); dc.DrawString("Congratulations! You reached a new level.", 365, 145) }
	if err := dc.LoadFontFace(findFont(), 18); err == nil { dc.SetRGB(0.78, 0.80, 0.87); dc.DrawString(fmt.Sprintf("Rank #%d    •    %d XP", d.Rank, d.XP), 365, 195) }
	return encodePNG(dc)
}

func fetchBackground(ctx context.Context, rawURL string) image.Image {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !(strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://")) { return nil }
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second); defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil); if err != nil { return nil }
	resp, err := http.DefaultClient.Do(req); if err != nil { return nil }; defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return nil }
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)); if err != nil || len(data) == 0 || len(data) > 10<<20 { return nil }
	img, _, err := image.Decode(bytes.NewReader(data)); if err != nil { return nil }
	return img
}

func fetchAvatar(ctx context.Context, url string) image.Image {
	if url == "" { return nil }
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second); defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil); if err != nil { return nil }
	resp, err := http.DefaultClient.Do(req); if err != nil { return nil }; defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return nil }
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20)); if err != nil { return nil }
	img, _, err := image.Decode(bytes.NewReader(data)); if err != nil { return nil }
	return img
}

func loadImage(path string) image.Image { if path == "" { return nil }; f, err := os.Open(path); if err != nil { return nil }; defer f.Close(); img, _, err := image.Decode(f); if err != nil { return nil }; return img }

func coverImage(src image.Image, width, height int) image.Image {
	b := src.Bounds(); sw, sh := b.Dx(), b.Dy(); if sw <= 0 || sh <= 0 { return src }
	scale := float64(width)/float64(sw); if v := float64(height)/float64(sh); v > scale { scale = v }
	nw, nh := int(float64(sw)*scale), int(float64(sh)*scale); resized := resizeImage(src, nw, nh)
	out := image.NewRGBA(image.Rect(0, 0, width, height)); x, y := (nw-width)/2, (nh-height)/2; draw.Draw(out, out.Bounds(), resized, image.Pt(x, y), draw.Src); return out
}

func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height)); b := src.Bounds(); sw, sh := b.Dx(), b.Dy()
	for y := 0; y < height; y++ { sy := b.Min.Y + y*sh/height; for x := 0; x < width; x++ { sx := b.Min.X + x*sw/width; dst.Set(x, y, src.At(sx, sy)) } }
	return dst
}

func resizeToSquare(src image.Image, size int) image.Image { return resizeImage(src, size, size) }
func encodePNG(dc *gg.Context) ([]byte, error) { var buf bytes.Buffer; if err := dc.EncodePNG(&buf); err != nil { return nil, fmt.Errorf("encode card: %w", err) }; return buf.Bytes(), nil }
func truncate(s string, max int) string { r := []rune(s); if len(r) <= max { return s }; return string(r[:max-1])+"…" }
