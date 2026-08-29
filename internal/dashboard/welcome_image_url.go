package dashboard

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// handleWelcomeImageURLSave stores an administrator-provided HTTPS image URL.
// The image itself is fetched by the card renderer when a member joins, so the
// dashboard does not need persistent file storage.
func (s *Server) handleWelcomeImageURLSave(w http.ResponseWriter, r *http.Request) {
	gid := guildIDFrom(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	raw := strings.TrimSpace(r.FormValue("welcome_image_url"))
	if raw != "" {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			http.Error(w, "image URL must be a valid HTTPS URL", http.StatusBadRequest)
			return
		}
		if len(raw) > 2048 {
			http.Error(w, "image URL is too long", http.StatusBadRequest)
			return
		}
	}
	cfg, err := s.store.GetWelcomeConfig(r.Context(), gid)
	if err != nil {
		http.Error(w, "failed to load welcome config", http.StatusInternalServerError)
		return
	}
	if raw == "" {
		cfg.JoinImageURL = nil
	} else {
		cfg.JoinImageURL = &raw
	}
	if err := s.store.UpsertWelcomeConfig(r.Context(), cfg); err != nil {
		http.Error(w, "failed to save welcome image URL", http.StatusInternalServerError)
		return
	}
	s.audit(r.Context(), r, "welcome.image_url.update", nil, map[string]any{"configured": raw != ""})
	http.Redirect(w, r, "/dashboard/"+gid+"/welcome", http.StatusSeeOther)
}

func (s *Server) handleWelcomeImageURLRemove(w http.ResponseWriter, r *http.Request) {
	gid := guildIDFrom(r)
	cfg, err := s.store.GetWelcomeConfig(r.Context(), gid)
	if err != nil {
		http.Error(w, "failed to load welcome config", http.StatusInternalServerError)
		return
	}
	cfg.JoinImageURL = nil
	if err := s.store.UpsertWelcomeConfig(r.Context(), cfg); err != nil {
		http.Error(w, "failed to remove welcome image URL", http.StatusInternalServerError)
		return
	}
	s.audit(r.Context(), r, "welcome.image_url.remove", nil, nil)
	http.Redirect(w, r, "/dashboard/"+gid+"/welcome", http.StatusSeeOther)
}

func validateWelcomeImageURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("invalid HTTPS image URL")
	}
	return nil
}
