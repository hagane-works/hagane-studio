package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
)

const sessionCookie = "haganeworks_session"

type Server struct {
	Mux *http.ServeMux

	Sessionless *http.ServeMux

	Tools *http.ServeMux

	host       string
	token      string
	toolsToken string
	web        fs.FS
}

func New(host string, web fs.FS, toolsToken string) (*Server, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return &Server{Mux: http.NewServeMux(), Sessionless: http.NewServeMux(), Tools: http.NewServeMux(),
		host: host, token: hex.EncodeToString(buf), toolsToken: toolsToken, web: web}, nil
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cross-Origin-Resource-Policy", "same-origin")

		header.Set("X-Hagane-Works", "1")
		if r.Host != s.host {
			Fail(w, http.StatusForbidden, errors.New("unexpected host"))
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+s.host {
			Fail(w, http.StatusForbidden, errors.New("cross-site request refused"))
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
			Fail(w, http.StatusForbidden, errors.New("cross-site request refused"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			header.Set("Cache-Control", "no-store")
			if handler, pattern := s.Sessionless.Handler(r); pattern != "" {
				handler.ServeHTTP(w, r)
				return
			}
			if _, pattern := s.Tools.Handler(r); pattern != "" {
				if !s.toolsAuthorized(r) {
					header.Set("WWW-Authenticate", "Bearer")
					Fail(w, http.StatusUnauthorized, errors.New("missing or wrong tools token (Authorization: Bearer <the tools-token file beside settings.json>)"))
					return
				}

				s.Tools.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(sessionCookie)
			if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(s.token)) != 1 {
				Fail(w, http.StatusUnauthorized, errors.New("session expired; reopen the window from Hagane Works"))
				return
			}
			s.Mux.ServeHTTP(w, r)
			return
		}
		s.serveWeb(w, r)
	})
}

func (s *Server) toolsAuthorized(r *http.Request) bool {
	presented, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return ok && s.toolsToken != "" && subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(s.toolsToken)) == 1
}

func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "" && name != "index.html" {
		if info, err := fs.Stat(s.web, name); err == nil && !info.IsDir() {

			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFileFS(w, r, s.web, name)
			return
		}
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    s.token,
		Path:     "/api/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.Header().Set("Cache-Control", "no-store")
	if _, err := fs.Stat(s.web, "index.html"); err != nil {
		http.Error(w, "The web UI is not built into this program. Run `npm run build` in web/, then rebuild.", http.StatusServiceUnavailable)
		return
	}
	http.ServeFileFS(w, r, s.web, "index.html")
}

func JSON(w http.ResponseWriter, status int, body any) {

	encoded, err := json.Marshal(body)
	if err != nil {
		log.Printf("encode response: %v", err)
		http.Error(w, `{"detail":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func Fail(w http.ResponseWriter, status int, err error) {
	JSON(w, status, map[string]string{"detail": err.Error()})
}

func ReadJSON(r *http.Request, limit int64, v any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, limit))
	decoder.UseNumber()
	return decoder.Decode(v)
}
