package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxWelcomeImageSize = 10 << 20

func (s *Server) handleWelcomeImageUpload(w http.ResponseWriter, r *http.Request) {
	gid := guildIDFrom(r)
	if err := r.ParseMultipartForm(maxWelcomeImageSize + (1 << 20)); err != nil {
		http.Error(w, "invalid upload", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("welcome_image")
	if err != nil { http.Error(w, "please choose an image", http.StatusBadRequest); return }
	defer file.Close()
	if header.Size > maxWelcomeImageSize { http.Error(w, "image must be 10 MB or smaller", http.StatusBadRequest); return }
	data, err := io.ReadAll(io.LimitReader(file, maxWelcomeImageSize+1))
	if err != nil || len(data) > maxWelcomeImageSize { http.Error(w, "could not read image", http.StatusBadRequest); return }
	cfg, err := s.store.GetWelcomeConfig(r.Context(), gid)
	if err != nil { http.Error(w, "failed to load welcome config", http.StatusInternalServerError); return }
	format, err := validateWelcomeImage(data)
	if err != nil { http.Error(w, "unsupported image: use PNG, JPEG or GIF", http.StatusBadRequest); return }
	path := filepath.Join("data", "welcome", gid+"."+format)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { http.Error(w, "failed to prepare image storage", http.StatusInternalServerError); return }
	if err := os.WriteFile(path, data, 0o644); err != nil { http.Error(w, "failed to save image", http.StatusInternalServerError); return }
	removeWelcomeImageFile(cfg.JoinImagePath, path)
	cfg.JoinImagePath = &path
	if err := s.store.UpsertWelcomeConfig(r.Context(), cfg); err != nil { _ = os.Remove(path); http.Error(w, "failed to save welcome settings", http.StatusInternalServerError); return }
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second); defer cancel()
	s.audit(ctx, r, "welcome.image.upload", nil, map[string]any{"format": format})
	http.Redirect(w, r, "/dashboard/"+gid+"/welcome", http.StatusSeeOther)
}

func (s *Server) handleWelcomeImageRemove(w http.ResponseWriter, r *http.Request) {
	gid := guildIDFrom(r)
	cfg, err := s.store.GetWelcomeConfig(r.Context(), gid)
	if err != nil { http.Error(w, "failed to load welcome config", http.StatusInternalServerError); return }
	removeWelcomeImageFile(cfg.JoinImagePath, "")
	cfg.JoinImagePath = nil
	if err := s.store.UpsertWelcomeConfig(r.Context(), cfg); err != nil { http.Error(w, "failed to save welcome settings", http.StatusInternalServerError); return }
	s.audit(r.Context(), r, "welcome.image.remove", nil, nil)
	http.Redirect(w, r, "/dashboard/"+gid+"/welcome", http.StatusSeeOther)
}

func (s *Server) handleWelcomeImage(w http.ResponseWriter, r *http.Request) {
	gid := guildIDFrom(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second); defer cancel()
	cfg, err := s.store.GetWelcomeConfig(ctx, gid)
	if err != nil || cfg.JoinImagePath == nil || *cfg.JoinImagePath == "" { http.NotFound(w, r); return }
	path := *cfg.JoinImagePath
	if filepath.Base(path) != gid+filepath.Ext(path) || filepath.Dir(filepath.Clean(path)) != filepath.Join("data", "welcome") { http.NotFound(w, r); return }
	http.ServeFile(w, r, path)
}

func validateWelcomeImage(data []byte) (string, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 2 || cfg.Height < 2 { return "", err }
	format = strings.ToLower(format)
	switch format { case "png", "jpeg", "gif": return format, nil; default: return "", fmt.Errorf("unsupported image format") }
}

func removeWelcomeImageFile(path *string, keep string) {
	if path == nil || *path == "" || *path == keep { return }
	_ = os.Remove(*path)
}
