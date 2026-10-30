package gameview

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"

	"haganeworks/internal/hoi4/guirender"
	"haganeworks/internal/hoi4/importassets"
	"haganeworks/internal/project"
)

const (
	spriteOverrideSchemaVersion = 1
	spriteBackupAssetType       = "gui_sprite_backup"
	maxSpriteTargetSide         = 8192
	maxSpriteTargetPixels       = int64(32 << 20)
	spriteSheetMaxFrames        = 256
	maxUploadBytes              = 10 * 1024 * 1024
)

func guiSpriteOverrideName(ref string) string {
	ref = strings.TrimSpace(ref)
	var b strings.Builder
	underscore := false
	for _, r := range ref {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-' {
			b.WriteRune(r)
			underscore = false
		} else if !underscore {
			b.WriteByte('_')
			underscore = true
		}
	}
	clean := b.String()
	if len(clean) > 80 {
		clean = clean[:80]
	}
	clean = strings.Trim(clean, "._")
	if clean == "" {
		clean = "sprite"
	}
	return clean + "_" + fmt.Sprintf("%x", sha1.Sum([]byte(ref)))[:10] + ".png"
}

type spriteRawBackup struct {
	DestPath            string  `json:"dest_path"`
	Content             string  `json:"content,omitempty"`
	IsBinary            bool    `json:"is_binary"`
	GenerateReplacePath bool    `json:"generate_replace_path,omitempty"`
	RawKind             string  `json:"raw_kind,omitempty"`
	TargetLayer         *string `json:"target_layer,omitempty"`
	TargetSHA256        *string `json:"target_sha256,omitempty"`
	Note                string  `json:"note,omitempty"`
	ContentSHA256       *string `json:"content_sha256,omitempty"`
	ContentOBSKey       *string `json:"content_obs_key,omitempty"`
	Source              string  `json:"source,omitempty"`
}

type spriteOverrideMetadata struct {
	SchemaVersion          int              `json:"schema_version"`
	Name                   string           `json:"name"`
	Created                bool             `json:"created,omitempty"`
	GFXPath                string           `json:"gfx_path,omitempty"`
	OriginalTextures       []string         `json:"original_textures"`
	OriginalTextureTokens  []string         `json:"original_texture_tokens"`
	GeneratedTexture       string           `json:"generated_texture"`
	GeneratedTextureSHA256 string           `json:"generated_texture_sha256,omitempty"`
	GFXManaged             bool             `json:"gfx_managed,omitempty"`
	GFXBaselineSHA256      string           `json:"gfx_baseline_sha256,omitempty"`
	GFXBackup              *spriteRawBackup `json:"gfx_backup,omitempty"`
	ScreenPicture          bool             `json:"screen_picture,omitempty"`
	Width                  int              `json:"width"`
	Height                 int              `json:"height"`
	Frames                 int              `json:"frames"`
	TextureBackup          *spriteRawBackup `json:"texture_backup,omitempty"`
}

func decodeSpriteMetadata(value any) (spriteOverrideMetadata, error) {
	var metadata spriteOverrideMetadata
	if value != nil {
		raw, err := json.Marshal(value)
		if err != nil {
			return metadata, err
		}
		if err := json.Unmarshal(raw, &metadata); err != nil {
			return metadata, err
		}
	}
	if metadata.OriginalTextures == nil {
		metadata.OriginalTextures = []string{}
	}
	if metadata.OriginalTextureTokens == nil {
		metadata.OriginalTextureTokens = []string{}
	}
	return metadata, nil
}

var errAssetMissing = errors.New("asset missing")

type spriteMutation struct {
	tx  *project.Tx
	raw *RawFiles
}

func (m *spriteMutation) assetRow(assetType, name string) project.Row {
	for _, row := range m.tx.Rows("project_obs_assets") {
		if str(row["asset_type"]) == assetType && str(row["gfx_name"]) == name {
			return row
		}
	}
	return nil
}

func (m *spriteMutation) resolve(basename string) (string, spriteOverrideMetadata, error) {
	row := m.assetRow("gui_sprite_override", basename)
	if row == nil || strings.TrimSpace(str(row["obs_key"])) == "" {
		return "", spriteOverrideMetadata{}, errAssetMissing
	}
	metadata, err := decodeSpriteMetadata(row["provenance"])
	return str(row["obs_key"]), metadata, err
}

func (m *spriteMutation) commit(basename, key string, metadata spriteOverrideMetadata) error {
	if strings.TrimSpace(basename) == "" || strings.TrimSpace(key) == "" {
		return errors.New("invalid GUI sprite override metadata")
	}
	provenance, err := jsonValue(metadata)
	if err != nil {
		return err
	}
	upsertAsset(m.tx, "gui_sprite_override", basename, key, provenance)
	backupKey := ""
	if metadata.TextureBackup != nil && metadata.TextureBackup.ContentOBSKey != nil {
		backupKey = strings.TrimSpace(*metadata.TextureBackup.ContentOBSKey)
	}
	if backupKey == "" {
		m.tx.Delete("project_obs_assets", func(row project.Row) bool {
			return str(row["asset_type"]) == spriteBackupAssetType && str(row["gfx_name"]) == basename
		})
		return nil
	}
	upsertAsset(m.tx, spriteBackupAssetType, basename, backupKey, map[string]any{})
	return nil
}

func (m *spriteMutation) remove(basename string) {
	m.tx.Delete("project_obs_assets", func(row project.Row) bool {
		kind := str(row["asset_type"])
		return (kind == "gui_sprite_override" || kind == spriteBackupAssetType) && str(row["gfx_name"]) == basename
	})
}

func (m *spriteMutation) overridesInGFX(gfxPath string) []project.Row {
	var out []project.Row
	for _, row := range m.tx.Rows("project_obs_assets") {
		if str(row["asset_type"]) != "gui_sprite_override" {
			continue
		}
		provenance, _ := row["provenance"].(map[string]any)
		if strings.EqualFold(strings.TrimSpace(str(provenance["gfx_path"])), strings.TrimSpace(gfxPath)) {
			out = append(out, row)
		}
	}
	return out
}

func (m *spriteMutation) activeGFXBaseline(gfxPath string) (spriteOverrideMetadata, bool) {
	var best project.Row
	bestHas := false
	for _, row := range m.overridesInGFX(gfxPath) {
		provenance, _ := row["provenance"].(map[string]any)
		has := str(provenance["gfx_baseline_sha256"]) != ""
		if best == nil || has && !bestHas || has == bestHas && str(row["created_at"]) < str(best["created_at"]) {
			best, bestHas = row, has
		}
	}
	if best == nil {
		return spriteOverrideMetadata{}, false
	}
	metadata, err := decodeSpriteMetadata(best["provenance"])
	return metadata, err == nil
}

func (m *spriteMutation) hasOtherActiveInGFX(gfxPath, exclude string) bool {
	for _, row := range m.overridesInGFX(gfxPath) {
		if str(row["gfx_name"]) != strings.TrimSpace(exclude) {
			return true
		}
	}
	return false
}

func withSpriteMutation(p *project.Project, fn func(*spriteMutation) error) error {
	return p.Write([]string{"project_raw_files", "project_obs_assets"}, func(tx *project.Tx) error {
		return fn(&spriteMutation{tx: tx, raw: NewRawFiles(tx)})
	})
}

func jsonValue(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	err = json.Unmarshal(raw, &out)
	return out, err
}

type spriteTarget struct {
	Index         *guirender.Index
	Width, Height int
	Frames        int
	Texture       string
}

func (s *Service) prepareSpriteTarget(r *http.Request, p *project.Project, name string) (spriteTarget, error) {
	index, found, err := s.loadIndex(r.Context(), p)
	if err != nil {
		return spriteTarget{}, err
	}
	if !found || index == nil {
		return spriteTarget{}, saveError{http.StatusNotFound, "GUI source snapshot unavailable"}
	}
	geometry, declared := index.SpriteTextureGeometry(name)
	if !declared {
		return spriteTarget{}, saveError{http.StatusNotFound, "no .gfx of the game or the mod declares " + name}
	}
	metadataGeometry := false
	if metadata, ok := storedSpriteMetadata(p, guiSpriteOverrideName(name)); ok &&
		usableSpriteMetadata(metadata, name) && strings.EqualFold(metadata.GeneratedTexture, geometry.Texture) &&
		metadata.Width > 0 && metadata.Height > 0 {
		geometry.Width, geometry.Height, geometry.Frames = metadata.Width, metadata.Height, max(1, metadata.Frames)
		metadataGeometry = true
	}

	if !metadataGeometry {
		if width, height, ok := projectRawTextureSize(p, index, geometry.Texture); ok {
			geometry.Width, geometry.Height = width, height
		}
	}
	if geometry.Width <= 0 || geometry.Height <= 0 {
		return spriteTarget{}, saveError{http.StatusUnprocessableEntity, "the target GUI sprite has no resolvable texture dimensions"}
	}
	if geometry.Width > maxSpriteTargetSide || geometry.Height > maxSpriteTargetSide ||
		int64(geometry.Width)*int64(geometry.Height) > maxSpriteTargetPixels {
		return spriteTarget{}, saveError{http.StatusUnprocessableEntity, "the target GUI sprite dimensions exceed the safe normalization limit"}
	}
	return spriteTarget{Index: index, Width: geometry.Width, Height: geometry.Height, Frames: max(1, geometry.Frames), Texture: geometry.Texture}, nil
}

func storedSpriteMetadata(p *project.Project, basename string) (spriteOverrideMetadata, bool) {
	rows, err := p.Rows("project_obs_assets")
	if err != nil {
		return spriteOverrideMetadata{}, false
	}
	for _, row := range rows {
		if str(row["asset_type"]) == "gui_sprite_override" && str(row["gfx_name"]) == basename && str(row["obs_key"]) != "" {
			metadata, err := decodeSpriteMetadata(row["provenance"])
			return metadata, err == nil
		}
	}
	return spriteOverrideMetadata{}, false
}

func projectRawTextureSize(p *project.Project, index *guirender.Index, texture string) (int, int, bool) {
	if strings.TrimSpace(texture) == "" {
		return 0, 0, false
	}
	if source, won := index.ProjectFileSource(texture); !won || !strings.HasPrefix(source, "project_raw_files:") {
		return 0, 0, false
	}
	rows, err := p.Rows("project_raw_files")
	if err != nil {
		return 0, 0, false
	}
	for _, row := range rows {
		file := rawFileFromRow(row)
		if !strings.EqualFold(file.DestPath, texture) || !file.IsBinary || file.ContentOBSKey == nil {
			continue
		}
		body, err := p.Blobs().Get(*file.ContentOBSKey, pictureLimit)
		if err != nil {
			return 0, 0, false
		}
		return guirender.TextureImageSize(file.DestPath, body)
	}
	return 0, 0, false
}

var safeCreatedSpriteName = regexp.MustCompile(`^GFX_[A-Za-z0-9_]{0,75}[A-Za-z0-9]$`)

func (s *Service) prepareCreatedSpriteTarget(r *http.Request, p *project.Project, name, width, height, frames string) (spriteTarget, error) {
	w, _ := strconv.Atoi(strings.TrimSpace(width))
	h, _ := strconv.Atoi(strings.TrimSpace(height))
	n := 1
	if strings.TrimSpace(frames) != "" {
		if n, _ = strconv.Atoi(strings.TrimSpace(frames)); n == 0 {
			n = -1
		}
	}
	if !safeCreatedSpriteName.MatchString(strings.TrimSpace(name)) {
		return spriteTarget{}, saveError{http.StatusUnprocessableEntity, "gfx_name must match GFX_[A-Za-z0-9_]*, end in a letter or digit and be at most 80 characters"}
	}
	if w <= 0 || h <= 0 || w > maxSpriteTargetSide || h > maxSpriteTargetSide || int64(w)*int64(h) > maxSpriteTargetPixels {
		return spriteTarget{}, saveError{http.StatusUnprocessableEntity, "width and height must describe a positive texture within the safe GUI sprite limit"}
	}
	if n < 1 || n > spriteSheetMaxFrames || w%n != 0 {
		return spriteTarget{}, saveError{http.StatusUnprocessableEntity, fmt.Sprintf("frames must be 1..%d and divide the full texture width exactly", spriteSheetMaxFrames)}
	}
	index, found, err := s.loadIndex(r.Context(), p)
	if err != nil {
		return spriteTarget{}, err
	}
	if !found || index == nil {
		return spriteTarget{}, saveError{http.StatusServiceUnavailable, "GUI source snapshot unavailable"}
	}
	return spriteTarget{Index: index, Width: w, Height: h, Frames: n}, nil
}

type uploadFit struct {
	Crop     importassets.Crop
	Fit      bool
	Stretch  bool
	Layout   string
	GridCols int
}

func (f uploadFit) isDefault() bool {
	return f.Crop == (importassets.Crop{}) && !f.Fit && !f.Stretch && (f.Layout == "" || f.Layout == "strip")
}

func parseUploadFit(r *http.Request) (uploadFit, error) {
	read := func(name string) (float64, error) {
		raw := strings.TrimSpace(r.FormValue(name))
		if raw == "" {
			return 0, nil
		}
		return strconv.ParseFloat(raw, 64)
	}
	var crop importassets.Crop
	var errs [4]error
	crop.X, errs[0] = read("crop_x")
	crop.Y, errs[1] = read("crop_y")
	crop.Width, errs[2] = read("crop_width")
	crop.Height, errs[3] = read("crop_height")
	if errors.Join(errs[:]...) != nil || crop.X < 0 || crop.Y < 0 || crop.Width < 0 || crop.Height < 0 {
		return uploadFit{}, saveError{http.StatusUnprocessableEntity, "crop_x, crop_y, crop_width and crop_height must be non-negative numbers"}
	}
	fit := uploadFit{Crop: crop, Layout: "strip"}
	switch strings.TrimSpace(r.FormValue("mode")) {
	case "", "fill":
	case "fit":
		fit.Fit = true
	case "stretch":
		fit.Stretch = true
	default:
		return uploadFit{}, saveError{http.StatusUnprocessableEntity, "mode must be fill, fit or stretch"}
	}
	switch strings.TrimSpace(r.FormValue("layout")) {
	case "", "strip":
	case "repeat":
		fit.Layout = "repeat"
	case "grid":
		cols, err := strconv.Atoi(strings.TrimSpace(r.FormValue("grid_cols")))
		if err != nil || cols < 1 || cols > spriteSheetMaxFrames {
			return uploadFit{}, saveError{http.StatusUnprocessableEntity, "grid_cols must be a positive number of columns"}
		}
		fit.Layout, fit.GridCols = "grid", cols
	default:
		return uploadFit{}, saveError{http.StatusUnprocessableEntity, "layout must be strip, repeat or grid"}
	}
	return fit, nil
}

func normalizeUpload(raw []byte, filename string, target spriteTarget, fit uploadFit) ([]byte, error) {
	if fit.isDefault() {

		width, height, err := importassets.ImageDimensions(raw, filename)
		if err != nil {
			return nil, saveError{http.StatusBadRequest, "could not decode uploaded image"}
		}
		if target.Frames > 1 && int64(width)*int64(target.Height) != int64(height)*int64(target.Width) {
			return nil, saveError{http.StatusUnprocessableEntity, fmt.Sprintf(
				"multi-frame GUI sprite requires a full %dx%d atlas with the same aspect ratio", target.Width, target.Height)}
		}
		out, err := importassets.NormalizePNGWithCropToSize(raw, filename, "gui_sprite_override", importassets.Crop{},
			importassets.Size{Width: target.Width, Height: target.Height})
		if err != nil {
			return nil, saveError{http.StatusBadRequest, "could not normalize uploaded image"}
		}
		return out, nil
	}
	img, err := importassets.DecodeImage(raw, filename)
	if err != nil {
		return nil, saveError{http.StatusBadRequest, "could not decode uploaded image"}
	}
	part := img.Bounds()
	if c := fit.Crop; c.Width > 0 && c.Height > 0 {
		b := img.Bounds()
		part = image.Rect(
			b.Min.X+int(math.Round(c.X)), b.Min.Y+int(math.Round(c.Y)),
			b.Min.X+int(math.Round(c.X+c.Width)), b.Min.Y+int(math.Round(c.Y+c.Height)),
		).Intersect(b)
		if part.Empty() {
			return nil, saveError{http.StatusUnprocessableEntity, "the crop is outside the picture"}
		}
	}
	frames := max(1, target.Frames)
	fw, fh := target.Width/frames, target.Height
	out := image.NewNRGBA(image.Rect(0, 0, target.Width, target.Height))
	place := func(dst, src image.Rectangle) {
		switch {
		case fit.Fit:
			dst = containRect(dst, src.Dx(), src.Dy())
		case !fit.Stretch:
			src = coverRect(src, dst.Dx(), dst.Dy())
		}
		xdraw.CatmullRom.Scale(out, dst, img, src, xdraw.Src, nil)
	}
	frame := func(f int) image.Rectangle { return image.Rect(f*fw, 0, (f+1)*fw, fh) }
	switch {
	case frames == 1:
		place(out.Bounds(), part)
	case fit.Layout == "repeat":
		for f := 0; f < frames; f++ {
			place(frame(f), part)
		}
	case fit.Layout == "grid":
		cols := min(fit.GridCols, frames)
		rows := (frames + cols - 1) / cols
		cw, ch := float64(part.Dx())/float64(cols), float64(part.Dy())/float64(rows)
		for f := 0; f < frames; f++ {
			x, y := float64(f%cols), float64(f/cols)
			cell := image.Rect(
				part.Min.X+int(math.Round(x*cw)), part.Min.Y+int(math.Round(y*ch)),
				part.Min.X+int(math.Round((x+1)*cw)), part.Min.Y+int(math.Round((y+1)*ch)),
			)
			if cell.Empty() {
				return nil, saveError{http.StatusUnprocessableEntity, "the picture is too small for that many frames"}
			}
			place(frame(f), cell)
		}
	default:
		if math.Abs(math.Log(float64(part.Dx())*float64(target.Height)/(float64(part.Dy())*float64(target.Width)))) > 0.02 {
			return nil, saveError{http.StatusUnprocessableEntity, fmt.Sprintf(
				"a %d-frame strip needs a %dx%d picture (or the same shape); choose one picture for every frame or a grid of frames instead",
				frames, target.Width, target.Height)}
		}
		xdraw.CatmullRom.Scale(out, out.Bounds(), img, part, xdraw.Src, nil)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, saveError{http.StatusBadRequest, "could not normalize uploaded image"}
	}
	return buf.Bytes(), nil
}

func containRect(dst image.Rectangle, w, h int) image.Rectangle {
	if w <= 0 || h <= 0 {
		return dst
	}
	scale := math.Min(float64(dst.Dx())/float64(w), float64(dst.Dy())/float64(h))
	cw := max(1, int(math.Round(float64(w)*scale)))
	ch := max(1, int(math.Round(float64(h)*scale)))
	left, top := dst.Min.X+(dst.Dx()-cw)/2, dst.Min.Y+(dst.Dy()-ch)/2
	return image.Rect(left, top, left+cw, top+ch)
}

func coverRect(bounds image.Rectangle, w, h int) image.Rectangle {
	bw, bh := bounds.Dx(), bounds.Dy()
	if bw*h > bh*w {
		cw := max(1, bh*w/h)
		left := bounds.Min.X + (bw-cw)/2
		return image.Rect(left, bounds.Min.Y, left+cw, bounds.Max.Y)
	}
	ch := max(1, bw*h/w)
	top := bounds.Min.Y + (bh-ch)/2
	return image.Rect(bounds.Min.X, top, bounds.Max.X, top+ch)
}

func writeMutationError(w http.ResponseWriter, operation string, err error) {
	var save saveError
	if errors.As(err, &save) {
		detail(w, save.status, save.detail)
		return
	}
	detail(w, http.StatusServiceUnavailable, "GUI sprite "+operation+" failed")
}

func (s *Service) spriteOverride(w http.ResponseWriter, r *http.Request, p *project.Project) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(2<<20))
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			detail(w, http.StatusUnprocessableEntity, "invalid multipart form")
			return
		}
		defer r.MultipartForm.RemoveAll()
		name = strings.TrimSpace(r.FormValue("name"))
	}
	if !validSelector(name) {
		detail(w, http.StatusUnprocessableEntity, "name is required")
		return
	}
	basename := guiSpriteOverrideName(name)
	if r.Method == http.MethodDelete {
		s.spriteRestore(w, r, p, name, basename)
		return
	}
	fit, err := parseUploadFit(r)
	if err != nil {
		writeMutationError(w, "normalize", err)
		return
	}

	create := r.FormValue("create") == "1"
	var target spriteTarget
	if create {
		target, err = s.prepareCreatedSpriteTarget(r, p, name, r.FormValue("width"), r.FormValue("height"), r.FormValue("frames"))
	} else {
		target, err = s.prepareSpriteTarget(r, p, name)
	}
	if err != nil {
		writeMutationError(w, "prepare", err)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		detail(w, http.StatusUnprocessableEntity, "file is required")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	_ = file.Close()
	if err != nil || len(raw) == 0 {
		detail(w, http.StatusBadRequest, "empty upload")
		return
	}
	if len(raw) > maxUploadBytes {
		detail(w, http.StatusRequestEntityTooLarge, "image exceeds the 10 MB size limit")
		return
	}
	pngBytes, err := normalizeUpload(raw, header.Filename, target, fit)
	if err != nil {
		writeMutationError(w, "normalize", err)
		return
	}
	sum := sha256.Sum256(pngBytes)
	blobs := p.Blobs()
	key, err := importassets.NamedAssetKey(blobs.EnvPrefix(), blobs.ProjectUUID(), "gui_sprite_override", basename, hex.EncodeToString(sum[:]))
	if err != nil {
		detail(w, http.StatusServiceUnavailable, "asset key unavailable")
		return
	}
	if !blobs.Exists(key) {
		if err := blobs.Put(key, "image/png", int64(len(pngBytes)), bytes.NewReader(pngBytes)); err != nil {
			detail(w, http.StatusServiceUnavailable, "the picture could not be stored; nothing was changed")
			return
		}
	}
	dds, err := spriteDDS(pngBytes)
	if err != nil {
		writeMutationError(w, "compile", err)
		return
	}
	ddsSHA, ddsKey, err := storeRawBlob(p, dds)
	if err != nil {
		writeMutationError(w, "compile", err)
		return
	}
	var compiled map[string]any
	err = withSpriteMutation(p, func(m *spriteMutation) error {
		_, previous, resolveErr := m.resolve(basename)
		if resolveErr != nil && !errors.Is(resolveErr, errAssetMissing) {
			return resolveErr
		}
		var metadata spriteOverrideMetadata
		var err error
		if create {
			compiled, metadata, err = compileCreatedSprite(m, target.Index, name, target.Width, target.Height, target.Frames,
				ddsSHA, ddsKey, previous, resolveErr == nil, true)
		} else {
			compiled, metadata, err = compileSpriteOverride(m, target.Index, target, name, ddsSHA, ddsKey, previous)
		}
		if err != nil {
			return err
		}
		return m.commit(basename, key, metadata)
	})
	s.Forget(p)
	if err != nil {
		writeMutationError(w, "compile", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "name": name, "width": target.Width, "height": target.Height,
		"frames": target.Frames, "override_path": "gui_sprite_overrides/" + basename, "export": compiled,
	})
}

func (s *Service) spriteRestore(w http.ResponseWriter, r *http.Request, p *project.Project, name, basename string) {
	below, found, err := s.loadImportedIndex(r.Context(), p)
	if err != nil || !found || below == nil {
		detail(w, http.StatusServiceUnavailable, "GUI source snapshot unavailable")
		return
	}
	err = withSpriteMutation(p, func(m *spriteMutation) error {
		_, metadata, resolveErr := m.resolve(basename)
		if errors.Is(resolveErr, errAssetMissing) {
			return nil
		}
		if resolveErr != nil {
			return resolveErr
		}
		if _, err := uncompileSpriteOverride(m, below, name, metadata); err != nil {
			return err
		}
		m.remove(basename)
		return nil
	})
	s.Forget(p)
	if err != nil {
		writeMutationError(w, "restore", err)
		return
	}
	resolved, width, height, frames := false, 0, 0, 1
	if target, err := s.prepareSpriteTarget(r, p, name); err == nil {
		width, height, frames, resolved = target.Width, target.Height, target.Frames, true
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name, "resolved": resolved, "width": width, "height": height, "frames": frames})
}
