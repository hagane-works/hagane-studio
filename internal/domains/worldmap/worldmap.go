package worldmap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/bmp"

	"haganeworks/internal/apiutil"
	"haganeworks/internal/domains"
	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/gamefiles"
	"haganeworks/internal/project"
)

const worldMapDefaultDate = "1936.1.1"

var worldMapDate = regexp.MustCompile(`^\d{1,4}\.\d{1,2}\.\d{1,2}(\.\d{1,2})?$`)

type handlers struct {
	deps domains.Deps
	svc  *gameview.Service
}

func Register(mux *http.ServeMux, deps domains.Deps) { RegisterWith(mux, deps, gameview.New(deps)) }

func RegisterWith(mux *http.ServeMux, deps domains.Deps, svc *gameview.Service) {
	h := &handlers{deps: deps, svc: svc}
	const prefix = "/api/go/projects/{id}/gui-render/world-map"
	routes := []struct {
		pattern string
		fn      func(http.ResponseWriter, *http.Request, *project.Project)
	}{
		{"GET " + prefix, h.worldMap},
		{"GET " + prefix + "/provinces.png", h.provincesPNG},
		{"GET " + prefix + "/legend", h.legend},
		{"GET " + prefix + "/rivers", h.riversRead},
		{"POST " + prefix + "/rivers", h.rivers},
		{"GET " + prefix + "/check", h.check},
		{"POST " + prefix + "/sketch-preview", h.sketchPreview},
		{"POST " + prefix + "/province", h.province},
		{"POST " + prefix + "/restate", h.restateRoute},
		{"POST " + prefix + "/terrain", h.terrain},
		{"POST " + prefix + "/flood", h.floodRoute},
		{"POST " + prefix + "/sink", h.sinkRoute},
		{"POST " + prefix + "/repair", h.repair},
		{"POST " + prefix + "/world-preview", h.worldRoute(true)},
		{"POST " + prefix + "/world", h.worldRoute(false)},
		{"POST " + prefix + "/land-preview", h.landRoute(true)},
		{"GET " + prefix + "/land", h.landRoute(false)},
		{"POST " + prefix + "/land", h.landRoute(false)},
		{"DELETE " + prefix + "/land", h.landRoute(false)},
		{"GET " + prefix + "/image", h.worldImageRead},
		{"POST " + prefix + "/image-import", h.worldImageImport},
		{"POST " + prefix + "/image-inspect", h.worldImageInspect},
		{"POST " + prefix + "/image-terrain", h.worldImageTerrain},
		{"POST " + prefix + "/image-countries", h.worldImageCountries},
		{"POST " + prefix + "/image-world-preview", h.worldImageWorldRoute(true)},
		{"POST " + prefix + "/image-world", h.worldImageWorldRoute(false)},
		{"POST " + prefix + "/names", h.names},
	}
	for _, route := range routes {
		mux.HandleFunc(route.pattern, h.route(route.fn))
	}

	mux.HandleFunc(prefix+"/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		guiHeaders(w)
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"detail": "Method Not Allowed"})
	})
}

func (h *handlers) route(fn func(http.ResponseWriter, *http.Request, *project.Project)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		guiHeaders(w)
		p, err := h.deps.Open(r)
		if err != nil {
			apiutil.Fail(w, err)
			return
		}
		fn(w, r, p)
	}
}

func (h *handlers) store(p *project.Project) *mapStore {
	return &mapStore{p: p, game: h.deps.Game}
}

func (h *handlers) gameIdentity() string {
	if h.deps.Game == nil {
		return ""
	}
	fsys, err := h.deps.Game()
	if err != nil || fsys == nil {
		return ""
	}
	return gamefiles.Identity(fsys)
}

func writeJSON(w http.ResponseWriter, status int, body any) { apiutil.JSON(w, status, body) }

func guiHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, noimageindex")
}

func (h *handlers) worldMap(w http.ResponseWriter, r *http.Request, p *project.Project) {
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = worldMapDefaultDate
	}
	if !worldMapDate.MatchString(date) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "date must look like 1936.1.1"})
		return
	}
	index, found, err := h.svc.Index(r.Context(), p)
	if err != nil {
		log.Printf("world map %s: %v", p.Dir(), err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "GUI source snapshot unavailable"})
		return
	}
	if !found || index == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "no game data for this project yet"})
		return
	}
	writeJSON(w, http.StatusOK, h.svc.InLanguage(r, index).WorldMap(date))
}

func (h *handlers) legend(w http.ResponseWriter, r *http.Request, p *project.Project) {
	writeJSON(w, http.StatusOK, map[string]any{"legend": worldMapLegend()})
}

func (h *handlers) provincesPNG(w http.ResponseWriter, r *http.Request, p *project.Project) {
	body, tag, err := h.projectProvincesPNG(p)
	if err == nil && body == nil {
		body, tag, err = h.gameProvincesPNG()
	}
	if err != nil {
		log.Printf("world map provinces image: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "原版地图图片还没放到服务器上(map/provinces.png)"})
		return
	}
	w.Header().Set("Content-Type", "image/png")

	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", tag)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Header.Get("If-None-Match") == tag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func bmpAsPNG(raw []byte) ([]byte, error) {
	picture, err := bmp.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&out, picture); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (h *handlers) projectProvincesPNG(p *project.Project) ([]byte, string, error) {
	sha, key, found, err := h.store(p).RawFileBlob("map/provinces.bmp")
	if err != nil || !found {
		return nil, "", err
	}
	tag := `"p` + sha[:min(16, len(sha))] + `"`
	if body, ok := projectMapImages.get(p.Dir(), sha); ok {
		return body, tag, nil
	}
	raw, err := p.Blobs().Get(key, 256<<20)
	if err != nil {
		return nil, "", err
	}
	body, err := bmpAsPNG(raw)
	if err != nil {
		return nil, "", fmt.Errorf("project map/provinces.bmp: %w", err)
	}
	projectMapImages.put(p.Dir(), sha, body)
	return body, tag, nil
}

func (h *handlers) gameProvincesPNG() ([]byte, string, error) {
	identity := h.gameIdentity()
	gameImage.Lock()
	defer gameImage.Unlock()
	if gameImage.body != nil && gameImage.game == identity {
		gameImage.used = time.Now()
		return gameImage.body, gameImage.tag, nil
	}
	if gameImage.err != nil && gameImage.game == identity && time.Since(gameImage.failedAt) < time.Minute {
		return nil, "", gameImage.err
	}
	gameImage.game, gameImage.body, gameImage.tag = identity, nil, ""
	body, err := h.loadGameProvincesPNG()
	if err != nil {
		gameImage.err, gameImage.failedAt = err, time.Now()
		return nil, "", err
	}
	sum := sha256.Sum256(body)
	gameImage.body, gameImage.tag, gameImage.err, gameImage.used = body, `"`+hex.EncodeToString(sum[:8])+`"`, nil, time.Now()
	startSweeper()
	return gameImage.body, gameImage.tag, nil
}

func (h *handlers) loadGameProvincesPNG() ([]byte, error) {
	raw, err := h.gameMapFile("provinces.bmp")
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("map/provinces.bmp is empty")
	}
	body, err := bmpAsPNG(raw)
	if err != nil {
		return nil, fmt.Errorf("decode map/provinces.bmp: %w", err)
	}
	return body, nil
}
