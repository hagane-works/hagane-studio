package gameview

import (
	"net/http"
	"strings"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/project"
)

func Register(mux *http.ServeMux, deps domains.Deps) { RegisterWith(mux, New(deps)) }

func RegisterWith(mux *http.ServeMux, svc *Service) {
	const prefix = "/api/go/projects/{id}/gui-render/"
	mux.HandleFunc("GET "+prefix+"{screen}", svc.handleGet)
	mux.HandleFunc("POST "+prefix+"{screen}", svc.handlePost)
	mux.HandleFunc("DELETE "+prefix+"{screen}", svc.handleDelete)
	mux.HandleFunc("GET "+prefix+"tech-canvas/layout", svc.handleTechCanvasLayout)
	mux.HandleFunc("POST "+prefix+"overlay/preview", svc.handleOverlayPreview)
	mux.HandleFunc("POST "+prefix+"overlay/validate", svc.handleOverlayValidate)
	mux.HandleFunc("POST "+prefix+"sprite/override", svc.handleSpriteOverride)
	mux.HandleFunc("DELETE "+prefix+"sprite/override", svc.handleSpriteOverride)
	mux.HandleFunc("GET /api/go/gui-vanilla/{path...}", svc.handleVanillaAsset)
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }

func detail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"detail": message})
}

func guiHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, noimageindex")
}

func (s *Service) open(w http.ResponseWriter, r *http.Request) (*project.Project, bool) {
	p, err := s.deps.Open(r)
	if err != nil {
		apiutil.Fail(w, err)
		return nil, false
	}
	return p, true
}

func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	screen := r.PathValue("screen")
	p, ok := s.open(w, r)
	if !ok {
		return
	}
	switch {
	case screen == "data-version":
		s.dataVersion(w, r, p)
	case screen == "text":
		s.text(w, r, p)
	case screen == "sprite":
		s.spriteRead(w, r, p)
	case screen == "scripted-gui-injection-points":
		s.injectionPoints(w, r, p)
	case screen == "font":
		s.fontRead(w, r, p)
	case screen == "overlay":
		s.overlayRead(w, r, p)
	default:
		s.read(w, r, p, screen)
	}
}

func (s *Service) handlePost(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	screen := r.PathValue("screen")
	p, ok := s.open(w, r)
	if !ok {
		return
	}
	switch screen {
	case "refresh":
		s.refresh(w, r, p)
	case "text":
		s.text(w, r, p)
	case "overlay":
		s.overlayWrite(w, r, p)
	case "export":
		s.bundleExport(w, r, p)
	default:
		detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	}
}

func (s *Service) handleDelete(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	p, ok := s.open(w, r)
	if !ok {
		return
	}
	if r.PathValue("screen") == "text" {
		s.text(w, r, p)
		return
	}
	detail(w, http.StatusMethodNotAllowed, "Method Not Allowed")
}

func (s *Service) handleTechCanvasLayout(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	p, ok := s.open(w, r)
	if !ok {
		return
	}
	s.read(w, r, p, "tech-canvas/layout")
}

func (s *Service) handleOverlayPreview(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	if p, ok := s.open(w, r); ok {
		s.overlayPreview(w, r, p)
	}
}

func (s *Service) handleOverlayValidate(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	if p, ok := s.open(w, r); ok {
		s.overlayValidate(w, r, p)
	}
}

func (s *Service) handleSpriteOverride(w http.ResponseWriter, r *http.Request) {
	guiHeaders(w)
	if p, ok := s.open(w, r); ok {
		s.spriteOverride(w, r, p)
	}
}

func validSelector(value string) bool {
	return value != "" && len(value) <= 512 && !strings.ContainsAny(value, "\r\n\x00")
}
