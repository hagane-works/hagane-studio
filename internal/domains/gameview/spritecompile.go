package gameview

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/imgcodec"
)

const (
	spriteGFXNote        = "游戏界面 · 换图(整份 .gfx 收进项目,只改换过图的 texturefile)"
	spriteTextureNote    = "游戏界面 · 换图的新图片"
	spriteCreatedGFXNote = "游戏界面 · Agent 新建 GFX sprite"
	screenPictureNote    = "游戏界面 · 换图(同路径盖住游戏的这张图片)"
	spriteTextureDir     = "gfx/interface/game_view/"
)

type saveError struct {
	status int
	detail string
}

func (e saveError) Error() string { return e.detail }

type spriteSourceIndex interface {
	SpriteSource(name string) (string, []byte, bool)
	ScreenPictureFile(name string) (string, bool)
}

func spriteTextureSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	slug := strings.Trim(b.String(), "_")
	if slug == "" {
		slug = "sprite"
	}
	return slug
}

func spriteTextureDest(name string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return spriteTextureDir + spriteTextureSlug(name) + "_" + hex.EncodeToString(sum[:])[:10] + ".dds"
}

func createdSpriteGFXDest(name string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return "interface/game_view_generated_" + spriteTextureSlug(name) + "_" + hex.EncodeToString(sum[:])[:10] + ".gfx"
}

func createdSpriteGFX(name, texture string, frames int) string {
	return fmt.Sprintf("spriteTypes = {\n\tspriteType = {\n\t\tname = %q\n\t\ttexturefile = %q\n\t\tnoOfFrames = %d\n\t}\n}\n", name, texture, max(1, frames))
}

func legacySpriteTextureDest(name string) string {
	return spriteTextureDir + spriteTextureSlug(name) + ".dds"
}

var overrideSuffix = regexp.MustCompile(`_[0-9a-f]{10}\.png$`)

func spriteNameOfOverride(basename string) (string, bool) {
	name := overrideSuffix.ReplaceAllString(basename, "")
	if name == basename || guiSpriteOverrideName(name) != basename {
		return "", false
	}
	return name, true
}

type span struct{ start, end int }

type scriptWord struct {
	text       string
	start, end int
	depth      int
}

func scriptWords(text string) []scriptWord {
	words := []scriptWord{}
	depth := 0
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == '#':
			for i < len(text) && text[i] != '\n' {
				i++
			}
		case c == '"':
			start := i
			i++
			for i < len(text) && text[i] != '"' {
				if text[i] == '\\' {
					i++
				}
				i++
			}
			i++
			words = append(words, scriptWord{text[start:min(i, len(text))], start, min(i, len(text)), depth})
		case c == '{':
			words = append(words, scriptWord{"{", i, i + 1, depth})
			depth++
			i++
		case c == '}':
			depth--
			words = append(words, scriptWord{"}", i, i + 1, depth})
			i++
		case c == '=' || c == ' ' || c == '\t' || c == '\r' || c == '\n':
			if c == '=' {
				words = append(words, scriptWord{"=", i, i + 1, depth})
			}
			i++
		default:
			start := i
			for i < len(text) && !strings.ContainsRune(" \t\r\n={}#\"", rune(text[i])) {
				i++
			}
			words = append(words, scriptWord{text[start:i], start, i, depth})
		}
	}
	return words
}

func gfxUnquote(value string) string { return strings.Trim(strings.TrimSpace(value), `"`) }

func spriteTextureSpans(text, name string) ([]span, []string) {
	words := scriptWords(text)
	var spans []span
	var values []string
	for i := 0; i+2 < len(words); i++ {
		if !strings.EqualFold(words[i].text, "name") || words[i+1].text != "=" || !strings.EqualFold(gfxUnquote(words[i+2].text), name) {
			continue
		}
		depth := words[i].depth
		open := -1
		for j := i - 1; j >= 0; j-- {
			if words[j].text == "{" && words[j].depth == depth-1 {
				open = j
				break
			}
		}
		if open < 2 || words[open-1].text != "=" || !strings.HasSuffix(strings.ToLower(words[open-2].text), "type") {
			continue
		}
		for j := open + 1; j+2 < len(words); j++ {
			if words[j].text == "}" && words[j].depth == depth-1 {
				break
			}
			if words[j].depth != depth || words[j+1].text != "=" {
				continue
			}
			key := strings.ToLower(words[j].text)
			if key == "texturefile" || key == "texturefile1" {
				spans = append(spans, span{words[j+2].start, words[j+2].end})
				values = append(values, gfxUnquote(words[j+2].text))
				break
			}
		}
	}
	return spans, values
}

func spriteTextureTokens(text string, spans []span) []string {
	out := make([]string, 0, len(spans))
	for _, item := range spans {
		if item.start < 0 || item.end < item.start || item.end > len(text) {
			return nil
		}
		out = append(out, text[item.start:item.end])
	}
	return out
}

func spriteTextureEdit(text, name, texture string) (string, error) {
	spans, _ := spriteTextureSpans(text, name)
	if len(spans) == 0 {
		return "", fmt.Errorf("%s has no texture file in this .gfx to repoint", name)
	}
	out := text
	for i := len(spans) - 1; i >= 0; i-- {
		out = out[:spans[i].start] + `"` + texture + `"` + out[spans[i].end:]
	}
	return out, nil
}

func spriteTextureRestore(text, name string, textures []string, quote bool) (string, error) {
	spans, _ := spriteTextureSpans(text, name)
	if len(spans) == 0 || len(spans) != len(textures) {
		return "", fmt.Errorf("%s declaration count changed while the override was active", name)
	}
	out := text
	for i := len(spans) - 1; i >= 0; i-- {
		value := textures[i]
		if quote {
			value = `"` + value + `"`
		}
		out = out[:spans[i].start] + value + out[spans[i].end:]
	}
	return out, nil
}

func spriteDDS(pngBytes []byte) ([]byte, error) {
	decoded, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	b := decoded.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("empty image")
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), decoded, b.Min, draw.Src)
	dds := imgcodec.EncodeDXT5DDS(nrgba.Pix, b.Dx(), b.Dy())
	if dds == nil {
		return nil, fmt.Errorf("could not encode the picture as DDS")
	}
	return dds, nil
}

func rawBackup(raw RawFile) *spriteRawBackup {
	return &spriteRawBackup{
		DestPath: raw.DestPath, Content: raw.Content, IsBinary: raw.IsBinary,
		GenerateReplacePath: raw.GenerateReplacePath, RawKind: raw.RawKind,
		TargetLayer: raw.TargetLayer, TargetSHA256: raw.TargetSHA256, Note: raw.Note,
		ContentSHA256: raw.ContentSHA256, ContentOBSKey: raw.ContentOBSKey, Source: raw.Source,
	}
}

func restoreRawBackup(m *spriteMutation, backup *spriteRawBackup) error {
	if backup == nil {
		return nil
	}
	_, err := m.raw.Create(RawFileInsert{
		DestPath: backup.DestPath, Content: backup.Content, IsBinary: backup.IsBinary,
		GenerateReplacePath: backup.GenerateReplacePath, RawKind: backup.RawKind,
		TargetLayer: backup.TargetLayer, TargetSHA256: backup.TargetSHA256, Note: backup.Note,
		ContentSHA256: backup.ContentSHA256, ContentOBSKey: backup.ContentOBSKey, Source: backup.Source,
	})
	return err
}

func usableSpriteMetadata(metadata spriteOverrideMetadata, name string) bool {
	base := metadata.SchemaVersion == spriteOverrideSchemaVersion &&
		strings.EqualFold(strings.TrimSpace(metadata.Name), strings.TrimSpace(name)) &&
		strings.TrimSpace(metadata.GeneratedTexture) != ""
	if !base || metadata.ScreenPicture {
		return base
	}
	if metadata.Created {
		return strings.TrimSpace(metadata.GFXPath) != "" && len(metadata.OriginalTextures) == 0
	}
	return strings.TrimSpace(metadata.GFXPath) != "" && len(metadata.OriginalTextures) > 0
}

func allTextureValuesEqual(values []string, wanted string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(wanted)) {
			return false
		}
	}
	return true
}

func gfxTextSHA256(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func usableGFXBaseline(metadata spriteOverrideMetadata) bool {
	want := strings.ToLower(strings.TrimSpace(metadata.GFXBaselineSHA256))
	if len(want) != sha256.Size*2 {
		return false
	}
	if metadata.GFXBackup == nil {
		return metadata.GFXManaged
	}
	backup := metadata.GFXBackup
	return !backup.IsBinary && strings.EqualFold(strings.TrimSpace(backup.DestPath), strings.TrimSpace(metadata.GFXPath)) &&
		strings.EqualFold(gfxTextSHA256(backup.Content), want)
}

func inheritGFXBaseline(dst *spriteOverrideMetadata, src spriteOverrideMetadata) {
	dst.GFXManaged, dst.GFXBaselineSHA256, dst.GFXBackup = src.GFXManaged, src.GFXBaselineSHA256, src.GFXBackup
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func gfxRowMetadataUnchanged(held RawFile, metadata spriteOverrideMetadata) bool {
	if held.IsBinary || !strings.EqualFold(held.DestPath, metadata.GFXPath) {
		return false
	}
	if metadata.GFXBackup == nil {
		return metadata.GFXManaged && held.RawKind == "verbatim_override" && held.Note == spriteGFXNote &&
			!held.GenerateReplacePath && held.TargetLayer == nil && held.TargetSHA256 == nil &&
			held.ContentSHA256 == nil && held.ContentOBSKey == nil && held.Source == "created"
	}
	backup := metadata.GFXBackup
	expectedSource := backup.Source
	if expectedSource == "imported" {
		expectedSource = "created"
	}
	return held.GenerateReplacePath == backup.GenerateReplacePath && held.RawKind == backup.RawKind &&
		held.Note == backup.Note && held.Source == expectedSource &&
		sameOptionalString(held.TargetLayer, backup.TargetLayer) &&
		sameOptionalString(held.TargetSHA256, backup.TargetSHA256) &&
		sameOptionalString(held.ContentSHA256, backup.ContentSHA256) &&
		sameOptionalString(held.ContentOBSKey, backup.ContentOBSKey)
}

func compileSpriteOverride(m *spriteMutation, index spriteSourceIndex, target spriteTarget, name, ddsSHA, ddsKey string, previous spriteOverrideMetadata) (map[string]any, spriteOverrideMetadata, error) {

	if previous.Created && usableSpriteMetadata(previous, name) {
		return compileCreatedSprite(m, target.Index, name, target.Width, target.Height, max(1, target.Frames), ddsSHA, ddsKey, previous, true, false)
	}
	files := m.raw.List()
	metadata := spriteOverrideMetadata{
		SchemaVersion: spriteOverrideSchemaVersion, Name: name,
		Width: target.Width, Height: target.Height, Frames: max(1, target.Frames),
		GeneratedTextureSHA256: ddsSHA,
	}
	file, body, ok := index.SpriteSource(name)
	if !ok {
		if picture, isPicture := index.ScreenPictureFile(name); isPicture {
			return compileScreenPicture(m, files, name, picture, ddsSHA, ddsKey, metadata, previous)
		}
		return nil, metadata, saveError{http.StatusNotFound, "no .gfx of the game or the mod declares " + name}
	}
	gfxDest, bad := SanitizeDestPath(file)
	if bad != "" {
		return nil, metadata, saveError{http.StatusUnprocessableEntity, bad}
	}
	if reserved := RejectReservedDest(gfxDest); reserved != "" {
		return nil, metadata, saveError{http.StatusUnprocessableEntity, reserved}
	}
	texture := spriteTextureDest(name)
	metadata.GFXPath, metadata.GeneratedTexture = gfxDest, texture

	raw, hasRaw := rawFileAt(files, gfxDest)
	current := string(body)
	if hasRaw {
		if raw.IsBinary {
			return nil, metadata, saveError{http.StatusConflict, "the current project .gfx source is binary"}
		}
		current = raw.Content
	}
	currentSpans, currentTextures := spriteTextureSpans(current, name)
	if len(currentTextures) == 0 {
		return nil, metadata, saveError{http.StatusConflict, "the current project .gfx no longer declares " + name}
	}
	previousForFile := usableSpriteMetadata(previous, name) && strings.EqualFold(previous.GFXPath, gfxDest)
	if previousForFile && strings.EqualFold(previous.GeneratedTexture, texture) {

		metadata.TextureBackup = previous.TextureBackup
	}
	if previousForFile && allTextureValuesEqual(currentTextures, previous.GeneratedTexture) {
		metadata.OriginalTextures = append([]string(nil), previous.OriginalTextures...)
		metadata.OriginalTextureTokens = append([]string(nil), previous.OriginalTextureTokens...)
	} else {
		if target.Texture != "" && !strings.EqualFold(strings.TrimSpace(target.Texture), strings.TrimSpace(currentTextures[0])) {
			return nil, metadata, saveError{http.StatusConflict, "the GUI sprite source changed; retry the replacement"}
		}
		metadata.OriginalTextures = append([]string(nil), currentTextures...)
		metadata.OriginalTextureTokens = spriteTextureTokens(current, currentSpans)
	}

	baselineFound := false
	if previousForFile {
		inheritGFXBaseline(&metadata, previous)
		baselineFound = true
	} else if sibling, found := m.activeGFXBaseline(gfxDest); found {
		inheritGFXBaseline(&metadata, sibling)
		baselineFound = true
	}
	if !baselineFound {
		metadata.GFXManaged = !hasRaw || raw.Note == spriteGFXNote
		if !hasRaw || raw.Note != spriteGFXNote {
			metadata.GFXBaselineSHA256 = gfxTextSHA256(current)
			if hasRaw {
				metadata.GFXBackup = rawBackup(raw)
			}
		}
	}
	text, err := spriteTextureEdit(current, name, texture)
	if err != nil {
		return nil, metadata, saveError{http.StatusUnprocessableEntity, err.Error()}
	}
	if err := GateRawContent(gfxDest, text, false); err != nil {
		return nil, metadata, saveError{http.StatusUnprocessableEntity, err.Error()}
	}
	if old, found := rawFileAt(files, texture); found {
		if old.Note != spriteTextureNote {
			metadata.TextureBackup = rawBackup(old)
		}
		if err := m.raw.Delete(old.ID); err != nil {
			return nil, metadata, err
		}
	}
	if _, err := m.raw.Create(RawFileInsert{
		DestPath: texture, IsBinary: true, RawKind: "verbatim_override", Note: spriteTextureNote,
		ContentSHA256: &ddsSHA, ContentOBSKey: &ddsKey,
	}); err != nil {
		return nil, metadata, err
	}
	if hasRaw {
		note := raw.Note
		if _, err := m.raw.Update(raw.ID, RawFileIntent{
			RawKind: raw.RawKind, DestPath: raw.DestPath, Content: text, GenerateReplacePath: raw.GenerateReplacePath,
			TargetLayer: raw.TargetLayer, TargetSHA256: raw.TargetSHA256,
		}, &note); err != nil {
			return nil, metadata, err
		}
	} else if _, err := m.raw.Create(RawFileInsert{
		DestPath: gfxDest, Content: text, RawKind: "verbatim_override", Note: spriteGFXNote,
	}); err != nil {
		return nil, metadata, err
	}
	return map[string]any{"gfx": gfxDest, "texture": texture}, metadata, nil
}

func compileScreenPicture(m *spriteMutation, files []RawFile, name, picture, ddsSHA, ddsKey string, metadata, previous spriteOverrideMetadata) (map[string]any, spriteOverrideMetadata, error) {
	dest, bad := SanitizeDestPath(picture)
	if bad != "" {
		return nil, metadata, saveError{http.StatusUnprocessableEntity, bad}
	}
	metadata.ScreenPicture = true
	metadata.GeneratedTexture = dest
	if usableSpriteMetadata(previous, name) && previous.ScreenPicture && strings.EqualFold(previous.GeneratedTexture, dest) {
		metadata.TextureBackup = previous.TextureBackup
	}
	if old, found := rawFileAt(files, dest); found {
		if old.Note != screenPictureNote {
			metadata.TextureBackup = rawBackup(old)
		}
		if err := m.raw.Delete(old.ID); err != nil {
			return nil, metadata, err
		}
	}
	if _, err := m.raw.Create(RawFileInsert{
		DestPath: dest, IsBinary: true, RawKind: "verbatim_override", Note: screenPictureNote,
		ContentSHA256: &ddsSHA, ContentOBSKey: &ddsKey,
	}); err != nil {
		return nil, metadata, err
	}
	return map[string]any{"texture": dest}, metadata, nil
}

func compileCreatedSprite(m *spriteMutation, index spriteSourceIndex, name string, width, height, frames int, ddsSHA, ddsKey string,
	previous spriteOverrideMetadata, previousFound, resize bool) (map[string]any, spriteOverrideMetadata, error) {
	gfxPath, texturePath := createdSpriteGFXDest(name), spriteTextureDest(name)
	metadata := spriteOverrideMetadata{
		SchemaVersion: spriteOverrideSchemaVersion, Name: name, Created: true, GFXPath: gfxPath,
		GeneratedTexture: texturePath, GeneratedTextureSHA256: ddsSHA, GFXManaged: true,
		Width: width, Height: height, Frames: frames, OriginalTextures: []string{}, OriginalTextureTokens: []string{},
	}
	files := m.raw.List()
	if previousFound {
		if !usableSpriteMetadata(previous, name) || !previous.Created ||
			!strings.EqualFold(previous.GFXPath, gfxPath) || !strings.EqualFold(previous.GeneratedTexture, texturePath) {
			return nil, metadata, saveError{http.StatusConflict, "the sprite name is already owned by another GUI declaration"}
		}
		if (!resize && (previous.Width != width || previous.Height != height)) || max(1, previous.Frames) != frames {
			return nil, metadata, saveError{http.StatusConflict, "the created sprite already exists with different geometry"}
		}
	} else if index != nil {
		if _, _, declared := index.SpriteSource(name); declared {
			return nil, metadata, saveError{http.StatusConflict, "the sprite already exists; upload a replacement instead"}
		}
	}
	for _, file := range files {
		if file.IsBinary {
			continue
		}
		if _, textures := spriteTextureSpans(file.Content, name); len(textures) > 0 && !strings.EqualFold(file.DestPath, gfxPath) {
			return nil, metadata, saveError{http.StatusConflict, "another project .gfx file already declares " + name}
		}
	}
	wantGFX := createdSpriteGFX(name, texturePath, frames)
	if held, found := rawFileAt(files, gfxPath); found {
		if !previousFound || held.IsBinary || held.Note != spriteCreatedGFXNote || held.Content != wantGFX {
			return nil, metadata, saveError{http.StatusConflict, "the generated sprite .gfx path is already occupied or was edited"}
		}
	} else {
		if previousFound {
			return nil, metadata, saveError{http.StatusConflict, "the generated sprite .gfx file is missing"}
		}
		if err := GateRawContent(gfxPath, wantGFX, false); err != nil {
			return nil, metadata, saveError{http.StatusUnprocessableEntity, err.Error()}
		}
		if _, err := m.raw.Create(RawFileInsert{DestPath: gfxPath, Content: wantGFX, RawKind: "verbatim_override", Note: spriteCreatedGFXNote}); err != nil {
			return nil, metadata, err
		}
	}
	if held, found := rawFileAt(files, texturePath); found {
		if !previousFound || !held.IsBinary || held.Note != spriteTextureNote {
			return nil, metadata, saveError{http.StatusConflict, "the generated sprite texture path is already occupied or was edited"}
		}

		if previous.GeneratedTextureSHA256 != "" && (held.ContentSHA256 == nil ||
			!strings.EqualFold(strings.TrimSpace(*held.ContentSHA256), strings.TrimSpace(previous.GeneratedTextureSHA256))) {
			return nil, metadata, saveError{http.StatusConflict, "the generated sprite texture was changed outside this workflow"}
		}
		if err := m.raw.Delete(held.ID); err != nil {
			return nil, metadata, err
		}
	}
	if _, err := m.raw.Create(RawFileInsert{
		DestPath: texturePath, IsBinary: true, RawKind: "verbatim_override", Note: spriteTextureNote,
		ContentSHA256: &ddsSHA, ContentOBSKey: &ddsKey,
	}); err != nil {
		return nil, metadata, err
	}
	return map[string]any{"gfx": gfxPath, "texture": texturePath}, metadata, nil
}

func uncompileSpriteOverride(m *spriteMutation, below spriteSourceIndex, name string, metadata spriteOverrideMetadata) (map[string]any, error) {
	files := m.raw.List()
	if !usableSpriteMetadata(metadata, name) {
		return uncompileLegacySpriteOverride(m, below, name, files)
	}
	if metadata.Created {
		return uncompileCreatedSprite(m, name, metadata, files), nil
	}
	out := map[string]any{}
	generated, generatedFound := rawFileAt(files, metadata.GeneratedTexture)
	expectedNote := spriteTextureNote
	if metadata.ScreenPicture {
		expectedNote = screenPictureNote
	}
	digestMatches := metadata.GeneratedTextureSHA256 == "" || generated.ContentSHA256 != nil &&
		strings.EqualFold(strings.TrimSpace(*generated.ContentSHA256), strings.TrimSpace(metadata.GeneratedTextureSHA256))
	if generatedFound && generated.Note == expectedNote && digestMatches {
		if err := m.raw.Delete(generated.ID); err != nil {
			return nil, err
		}
		out["texture_removed"] = metadata.GeneratedTexture
		if err := restoreRawBackup(m, metadata.TextureBackup); err != nil {
			return nil, err
		}
		if metadata.TextureBackup != nil {
			out["texture_restored"] = metadata.TextureBackup.DestPath
		}
	} else if !generatedFound && metadata.TextureBackup != nil {
		if err := restoreRawBackup(m, metadata.TextureBackup); err != nil {
			return nil, err
		}
		out["texture_restored"] = metadata.TextureBackup.DestPath
	}
	if metadata.ScreenPicture {
		return out, nil
	}
	held, ok := rawFileAt(files, metadata.GFXPath)
	if !ok || held.IsBinary {
		return out, nil
	}
	_, currentTextures := spriteTextureSpans(held.Content, name)
	if !allTextureValuesEqual(currentTextures, metadata.GeneratedTexture) {

		return out, nil
	}
	var text string
	var err error
	if len(metadata.OriginalTextureTokens) == len(currentTextures) {
		text, err = spriteTextureRestore(held.Content, name, metadata.OriginalTextureTokens, false)
	} else {
		text, err = spriteTextureRestore(held.Content, name, metadata.OriginalTextures, true)
	}
	if err != nil {
		return nil, err
	}
	hasOtherActive := m.hasOtherActiveInGFX(metadata.GFXPath, guiSpriteOverrideName(name))
	if !hasOtherActive && usableGFXBaseline(metadata) &&
		strings.EqualFold(gfxTextSHA256(text), metadata.GFXBaselineSHA256) && gfxRowMetadataUnchanged(held, metadata) {
		if err := m.raw.Delete(held.ID); err != nil {
			return nil, err
		}
		if metadata.GFXBackup != nil {
			if err := restoreRawBackup(m, metadata.GFXBackup); err != nil {
				return nil, err
			}
			out["gfx_restored"] = metadata.GFXPath
		} else {
			out["gfx_removed"] = metadata.GFXPath
		}
		return out, nil
	}

	if !hasOtherActive && metadata.GFXManaged && metadata.GFXBaselineSHA256 == "" && gfxRowMetadataUnchanged(held, metadata) {
		if file, original, declared := below.SpriteSource(name); declared && strings.EqualFold(file, metadata.GFXPath) && text == string(original) {
			if err := m.raw.Delete(held.ID); err != nil {
				return nil, err
			}
			out["gfx_removed"] = metadata.GFXPath
			return out, nil
		}
	}
	note := held.Note
	if _, err := m.raw.Update(held.ID, RawFileIntent{
		RawKind: held.RawKind, DestPath: held.DestPath, Content: text, GenerateReplacePath: held.GenerateReplacePath,
		TargetLayer: held.TargetLayer, TargetSHA256: held.TargetSHA256,
	}, &note); err != nil {
		return nil, err
	}
	out["gfx"] = metadata.GFXPath
	return out, nil
}

func uncompileCreatedSprite(m *spriteMutation, name string, metadata spriteOverrideMetadata, files []RawFile) map[string]any {
	out := map[string]any{}
	texture, textureFound := rawFileAt(files, metadata.GeneratedTexture)
	textureOwned := textureFound && texture.Note == spriteTextureNote &&
		metadata.GeneratedTextureSHA256 != "" && texture.ContentSHA256 != nil &&
		strings.EqualFold(strings.TrimSpace(*texture.ContentSHA256), strings.TrimSpace(metadata.GeneratedTextureSHA256))
	gfx, gfxFound := rawFileAt(files, metadata.GFXPath)
	gfxOwned := gfxFound && !gfx.IsBinary && gfx.Note == spriteCreatedGFXNote &&
		gfx.Content == createdSpriteGFX(name, metadata.GeneratedTexture, metadata.Frames)
	if (textureFound && !textureOwned) || (gfxFound && !gfxOwned) {
		return out
	}
	if textureOwned && m.raw.Delete(texture.ID) == nil {
		out["texture_removed"] = metadata.GeneratedTexture
	}
	if gfxOwned && m.raw.Delete(gfx.ID) == nil {
		out["gfx_removed"] = metadata.GFXPath
	}
	return out
}

func uncompileLegacySpriteOverride(m *spriteMutation, below spriteSourceIndex, name string, files []RawFile) (map[string]any, error) {
	out := map[string]any{}
	if picture, ok := below.ScreenPictureFile(name); ok {
		if held, found := rawFileAt(files, picture); found && held.Note == screenPictureNote {
			if err := m.raw.Delete(held.ID); err != nil {
				return nil, err
			}
			out["texture_removed"] = picture
		}
		return out, nil
	}
	file, original, ok := below.SpriteSource(name)
	if !ok {
		return out, nil
	}
	gfxDest, bad := SanitizeDestPath(file)
	if bad != "" {
		return nil, saveError{http.StatusUnprocessableEntity, bad}
	}
	held, found := rawFileAt(files, gfxDest)
	if !found || held.IsBinary {
		return out, nil
	}
	if held.Note != spriteGFXNote {
		return nil, saveError{http.StatusConflict, "this legacy replacement has no reversible project .gfx provenance"}
	}
	spans, values := spriteTextureSpans(string(original), name)
	if len(values) == 0 {
		return out, nil
	}
	text, err := spriteTextureRestore(held.Content, name, spriteTextureTokens(string(original), spans), false)
	if err != nil {
		return nil, err
	}
	if texture, exists := rawFileAt(files, legacySpriteTextureDest(name)); exists && texture.Note == spriteTextureNote {
		if err := m.raw.Delete(texture.ID); err != nil {
			return nil, err
		}
		out["texture_removed"] = texture.DestPath
	}
	if text == string(original) {
		if err := m.raw.Delete(held.ID); err != nil {
			return nil, err
		}
		out["gfx_removed"] = gfxDest
		return out, nil
	}
	note := held.Note
	_, err = m.raw.Update(held.ID, RawFileIntent{
		RawKind: held.RawKind, DestPath: held.DestPath, Content: text, GenerateReplacePath: held.GenerateReplacePath,
		TargetLayer: held.TargetLayer, TargetSHA256: held.TargetSHA256,
	}, &note)
	return out, err
}
