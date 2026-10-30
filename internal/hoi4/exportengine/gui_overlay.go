package exportengine

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"haganeworks/internal/hoi4/emit/locout"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/importsvc"
	"haganeworks/internal/hoi4/paradox"
)

const guiSnapshotLimit = int64(512 << 20)

type guiEdit struct {
	Target  string         `json:"target"`
	Op      string         `json:"op"`
	Payload map[string]any `json:"payload"`
}

type guiOverlayRow struct {
	File   string    `json:"file"`
	Source string    `json:"source"`
	Edits  []guiEdit `json:"edits"`

	ScaffoldHash string `json:"scaffold_hash"`
}

const (
	guiVanillaScaffoldAsset = "gui_vanilla_scaffold"
	guiVanillaScaffoldLimit = 16 << 20
)

func guiOverlayScaffold(overlay guiOverlayRow, rel string, sources map[string][]byte, assets map[string]string, resolve BlobResolver) []byte {
	imported := sources[rel]
	want := strings.ToLower(strings.TrimSpace(overlay.ScaffoldHash))
	if imported != nil && (want == "" || sha256Hex(imported) == want) {
		return imported
	}
	if key := assets[guiVanillaScaffoldAsset+"\x00"+want]; want != "" && key != "" {
		if pinned, err := resolve(key, guiVanillaScaffoldLimit); err == nil && sha256Hex(pinned) == want {
			return pinned
		}
	}
	return imported
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func nativeGUIOverlays(snapshot Snapshot, modDir string, resolve BlobResolver, sink *imageSink, includeUnedited bool) ([]Entry, error) {
	if len(snapshot.Tables["gui_overlays"]) == 0 {
		return nil, nil
	}
	assets, err := projectOBSAssets(snapshot)
	if err != nil {
		return nil, err
	}
	snapshotKey := assets["gui_source\x00snapshot.zip"]
	if resolve == nil {
		sink.say("gui.overlays_skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.no_access"}})
		return nil, nil
	}

	sources := map[string][]byte{}
	if snapshotKey != "" {
		rawZip, err := resolve(snapshotKey, guiSnapshotLimit)
		if err != nil {
			sink.say("gui.overlays_skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.snapshot_unread"}})
			return nil, nil
		}

		if isManifestOnlyBlob(rawZip) {
			return nativeGUIOverlayFileIndex(snapshot, modDir, assets, true)
		}
		if sources, err = readSafeZip(rawZip, guiSnapshotLimit); err != nil {
			sink.say("gui.overlays_skipped", exportmsg.Args{"reason": exportmsg.Message{Key: "gui.reason.snapshot_broken"}})
			return nil, nil
		}
	} else if probe := firstGUIVanillaScaffoldKey(assets); probe != "" {
		if blob, err := resolve(probe, guiVanillaScaffoldLimit); err == nil && isManifestOnlyBlob(blob) {
			return nativeGUIOverlayFileIndex(snapshot, modDir, assets, false)
		}
	}
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	langs := emitmisc.BuildExportLangTriples(stringList(project, "export_languages"))
	var out []Entry
	for _, raw := range snapshot.Tables["gui_overlays"] {
		var overlay guiOverlayRow
		if err := json.Unmarshal(raw, &overlay); err != nil {
			return nil, fmt.Errorf("decode gui overlay: %w", err)
		}
		if len(overlay.Edits) == 0 && !includeUnedited {
			continue
		}
		rel, err := safeGUIPath(overlay.File)
		if err != nil {
			return nil, err
		}
		source := guiOverlayScaffold(overlay, rel, sources, assets, resolve)
		if source == nil {

			sink.say("gui.overlay_no_scaffold", exportmsg.Args{"file": rel})
			continue
		}
		ast, recoveries, err := paradox.Parse(string(source))
		if err != nil || !guiRecoveriesWritable(recoveries) {

			reason := fmt.Sprint(err)
			if err == nil {
				reason = recoveries[0].Code
			}
			sink.say("gui.overlay_unreadable", exportmsg.Args{"file": rel, "reason": reason})
			continue
		}
		var sprites []emitmisc.SpritePair
		var spriteEntries []Entry
		locByLang := map[string]map[string]string{}

		var orphans []string
		for _, edit := range overlay.Edits {
			node, parent, index := resolveGUINode(ast, edit.Target)
			if node == nil {
				orphans = append(orphans, guiOrphanLabel(edit))
			}
			if edit.Op == "add_node" {
				if node != nil {
					if rawNode, ok := edit.Payload["node"].(map[string]any); ok {
						child := guiEntryFromJSON(rawNode)
						insert := len(node.Block.Entries)
						if value, ok := edit.Payload["index"].(float64); ok {

							wanted, seen := int(value), 0
							for i, entry := range node.Block.Entries {
								if entry.Block != nil && entry.Key != nil && strings.HasSuffix(strings.ToLower(*entry.Key), "type") {
									if seen == wanted {
										insert = i
										break
									}
									seen++
								}
							}
						}
						node.Block.Entries = append(node.Block.Entries, nil)
						copy(node.Block.Entries[insert+1:], node.Block.Entries[insert:])
						node.Block.Entries[insert] = child
					}
				}
				continue
			}
			if edit.Op == "delete_node" {
				if node != nil && parent != nil {
					parent.Entries = append(parent.Entries[:index], parent.Entries[index+1:]...)
				}
				continue
			}
			if node == nil {
				continue
			}
			switch edit.Op {
			case "set_text":
				text := guiOverlayTextValue(edit.Payload)
				if guiOverlayTextSupplied(edit.Payload) {
					setGUIString(node, "text", text)
				}
				key := strings.TrimSpace(guiPayloadString(edit.Payload, "key"))
				if key != "" {
					for _, lang := range langs {
						value := guiOverlayLocalizationValue(edit.Payload, lang.Code)
						if value == "" {
							continue
						}
						emitKey := key
						if isNew, ok := edit.Payload["is_new"].(bool); !ok || isNew {
							emitKey = "GUISUB_" + key
						}
						if locByLang[lang.Code] == nil {
							locByLang[lang.Code] = map[string]string{}
						}
						locByLang[lang.Code][emitKey] = value
					}
				}
			case "swap_sprite":
				if engineSwappedGUITarget(edit.Target) {
					sink.say("gui.swap_engine_picture", exportmsg.Args{"file": rel, "target": edit.Target})
					continue
				}
				gfxRef := strings.TrimSpace(guiPayloadString(edit.Payload, "gfx_ref"))
				basename := path.Base(strings.ReplaceAll(guiPayloadString(edit.Payload, "override_file"), "\\", "/"))
				if basename == "." || basename == "" {

					if gfxRef != "" {
						setGUIImage(node, gfxRef)
					}
					continue
				}
				key := assets["gui_sprite_override\x00"+basename]
				if key == "" {

					sink.say("gui.swap_no_upload", exportmsg.Args{"file": rel, "target": edit.Target, "picture": basename})
					continue
				}
				slug := guiSpriteSlug(basename)
				gfxName := "GFX_mysub_" + slug
				blob, err := resolve(key, 32<<20)
				if err != nil {
					return nil, fmt.Errorf("resolve GUI sprite %q: %w", basename, err)
				}
				dds, err := imageBytesToDDS(blob, basename)
				if err != nil {
					return nil, fmt.Errorf("encode GUI sprite %q: %w", basename, err)
				}
				ddsPath := "gfx/interface/" + slug + ".dds"
				setGUIImage(node, gfxName)
				sprites = append(sprites, emitmisc.SpritePair{Name: gfxName, Path: ddsPath})
				spriteEntries = append(spriteEntries, Entry{Path: modDir + "/" + ddsPath, Bytes: dds, Origin: "generated", SourceEntity: "gui_overlay_sprites"})
			case "set_layout":
				setGUIPair(node, "position", "x", edit.Payload["x"], "y", edit.Payload["y"])
				setGUIPair(node, "size", "width", edit.Payload["width"], "height", edit.Payload["height"])
			case "hide_node":
				hideGUINode(node)
			case "set_orientation":
				setGUIString(node, "orientation", guiPayloadString(edit.Payload, "value"))
			case "set_origo":
				value := guiPayloadString(edit.Payload, "value")
				if value == "" {
					deleteGUIKey(node, "origo")
				} else {
					setGUIString(node, "origo", value)
				}
			}
		}
		if len(orphans) > 0 {
			sink.note(guiOrphanNote(rel, orphans, sink.lang()))
		}
		out = append(out, Entry{Path: modDir + "/" + rel, Bytes: []byte(importsvc.ToScript(ast, 0) + "\n"), Origin: "generated", SourceEntity: "gui_overlays"})
		if len(sprites) > 0 {
			out = append(out, Entry{Path: modDir + "/interface/" + strings.TrimSuffix(path.Base(rel), path.Ext(rel)) + "_sprites.gfx", Bytes: []byte(emitmisc.EmitSpriteBlock(dedupeSprites(sprites))), Origin: "generated", SourceEntity: "gui_overlay_sprites"})
			out = append(out, spriteEntries...)
		}
		stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		if paired := sources["common/scripted_guis/"+stem+".txt"]; paired != nil {
			out = append(out, Entry{Path: modDir + "/common/scripted_guis/" + stem + ".txt", Bytes: paired, Origin: "imported", SourceEntity: "gui_overlay_scripted_guis"})
		}
		for _, lang := range langs {
			values := locByLang[lang.Code]
			if len(values) == 0 {
				continue
			}
			keys := make([]string, 0, len(values))
			for key := range values {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			lines := []string{lang.Suffix + ":"}
			for _, key := range keys {
				lines = append(lines, " "+key+": \""+escapeYML(values[key])+"\"")
			}
			out = append(out, Entry{Path: modDir + "/localisation/" + lang.Subdir + "/" + stem + "_gui_submod_" + lang.Suffix + ".yml", Bytes: []byte("\xEF\xBB\xBF" + strings.Join(lines, "\n") + "\n"), Origin: "generated", SourceEntity: "gui_overlay_localisation"})
		}
	}
	return out, nil
}

func BuildNativeGUIBundle(snapshot Snapshot, resolve BlobResolver) ([]Entry, []string, error) {
	sink := &imageSink{}
	entries, err := nativeGUIOverlays(snapshot, "bundle", resolve, sink, true)
	if err != nil {
		return nil, sink.warnings, err
	}
	for i := range entries {
		entries[i].Path = strings.TrimPrefix(entries[i].Path, "bundle/")
	}
	return entries, sink.warnings, nil
}

func firstGUIVanillaScaffoldKey(assets map[string]string) string {
	keys := make([]string, 0, len(assets))
	for name, key := range assets {
		if strings.HasPrefix(name, guiVanillaScaffoldAsset+"\x00") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func nativeGUIOverlayFileIndex(snapshot Snapshot, modDir string, assets map[string]string, hasImportedSnapshot bool) ([]Entry, error) {
	project, err := emitmisc.DecodeEntityJSON(snapshot.Project)
	if err != nil {
		return nil, err
	}
	langs := emitmisc.BuildExportLangTriples(stringList(project, "export_languages"))
	var out []Entry
	for _, raw := range snapshot.Tables["gui_overlays"] {
		var overlay guiOverlayRow
		if err := json.Unmarshal(raw, &overlay); err != nil {
			return nil, fmt.Errorf("decode gui overlay: %w", err)
		}
		if len(overlay.Edits) == 0 {
			continue
		}

		pinned := assets[guiVanillaScaffoldAsset+"\x00"+strings.ToLower(strings.TrimSpace(overlay.ScaffoldHash))] != ""
		if strings.EqualFold(strings.TrimSpace(overlay.Source), "vanilla") && !pinned {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(overlay.Source), "vanilla") && !hasImportedSnapshot && !pinned {
			continue
		}
		rel, err := safeGUIPath(overlay.File)
		if err != nil {
			return nil, err
		}
		out = append(out, Entry{Path: modDir + "/" + rel, Origin: "generated", SourceEntity: "gui_overlays"})

		stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
		hasSprite := false
		locLangs := map[string]bool{}
		for _, edit := range overlay.Edits {
			switch edit.Op {
			case "swap_sprite":
				basename := path.Base(strings.ReplaceAll(guiPayloadString(edit.Payload, "override_file"), "\\", "/"))
				if basename == "." || basename == "" || assets["gui_sprite_override\x00"+basename] == "" {
					continue
				}
				hasSprite = true
				slug := guiSpriteSlug(basename)
				out = append(out, Entry{Path: modDir + "/gfx/interface/" + slug + ".dds", Origin: "generated", SourceEntity: "gui_overlay_sprites"})
			case "set_text":
				key := strings.TrimSpace(guiPayloadString(edit.Payload, "key"))
				if key == "" {
					continue
				}
				fallback := guiPayloadString(edit.Payload, "text")
				if fallback == "" {
					fallback = guiLangText(edit.Payload, "en")
				}
				for _, lang := range langs {
					if guiLangText(edit.Payload, lang.Code) != "" || fallback != "" {
						locLangs[lang.Code] = true
					}
				}
			}
		}
		if hasSprite {
			out = append(out, Entry{Path: modDir + "/interface/" + stem + "_sprites.gfx", Origin: "generated", SourceEntity: "gui_overlay_sprites"})
		}
		for _, lang := range langs {
			if locLangs[lang.Code] {
				out = append(out, Entry{Path: modDir + "/localisation/" + lang.Subdir + "/" + stem + "_gui_submod_" + lang.Suffix + ".yml", Origin: "generated", SourceEntity: "gui_overlay_localisation"})
			}
		}
	}
	return out, nil
}

func projectOBSAssets(snapshot Snapshot) (map[string]string, error) {
	out := map[string]string{}
	for _, raw := range snapshot.Tables["project_obs_assets"] {
		var row struct {
			AssetType string `json:"asset_type"`
			GFXName   string `json:"gfx_name"`
			OBSKey    string `json:"obs_key"`
		}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}
		if row.OBSKey != "" {
			out[row.AssetType+"\x00"+row.GFXName] = row.OBSKey
		}
	}
	return out, nil
}

func readSafeZip(raw []byte, limit int64) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	var total int64
	for _, f := range zr.File {
		n := strings.ReplaceAll(f.Name, "\\", "/")
		if _, err := SafePath(n); err != nil {
			return nil, err
		}
		if f.FileInfo().IsDir() {
			continue
		}
		total += int64(f.UncompressedSize64)
		if total > limit {
			return nil, fmt.Errorf("expanded archive exceeds limit")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, limit+1))
		rc.Close()
		if err != nil {
			return nil, err
		}
		out[n] = b
	}
	return out, nil
}
func safeGUIPath(value string) (string, error) {
	v := strings.Trim(strings.ReplaceAll(value, "\\", "/"), "/")
	if !strings.HasPrefix(strings.ToLower(v), "interface/") {
		v = "interface/" + path.Base(v)
	}
	if !strings.HasSuffix(strings.ToLower(v), ".gui") {
		v += ".gui"
	}
	if _, err := SafePath(v); err != nil {
		return "", err
	}
	return v, nil
}

func guiBlocks(b *paradox.Block) []*paradox.Entry {
	var out []*paradox.Entry
	for _, e := range b.Entries {
		if e.Key != nil && e.Block != nil {
			key := strings.ToLower(*e.Key)
			if key == "guitypes" {
				out = append(out, guiBlocks(e.Block)...)
			} else if key == "containerwindowtype" || key == "windowtype" {
				out = append(out, e)
			}
		}
	}
	return out
}
func guiName(e *paradox.Entry) string {
	if e == nil || e.Block == nil {
		return ""
	}
	for _, x := range e.Block.Entries {
		if x.Key != nil && strings.EqualFold(*x.Key, "name") && x.Block == nil {
			return x.Scalar.StrValue()
		}
	}
	return ""
}
func guiChildren(e *paradox.Entry) []*paradox.Entry {
	if e == nil || e.Block == nil {
		return nil
	}
	var out []*paradox.Entry
	for _, x := range e.Block.Entries {
		if x.Key != nil && x.Block != nil {
			key := strings.ToLower(*x.Key)
			if strings.HasSuffix(key, "type") || key == "background" {
				out = append(out, x)
			}
		}
	}
	return out
}

func guiOrphanLabel(edit guiEdit) string {
	op := strings.TrimSpace(edit.Op)
	if op == "" {
		op = "edit"
	}
	target := strings.TrimSpace(edit.Target)
	if target == "" {
		target = "(no target)"
	}
	return op + " on " + target
}

func guiOrphanNote(rel string, orphans []string, msg exportmsg.Printer) string {
	return msg.Text("gui.orphan_edits", exportmsg.Args{"file": rel, "count": len(orphans), "targets": joinCapped(orphans, 3, msg)})
}

func resolveGUINode(root *paradox.Block, target string) (node *paradox.Entry, parent *paradox.Block, index int) {
	parts := strings.FieldsFunc(target, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return nil, nil, -1
	}
	candidates := guiBlocks(root)
	for depth, name := range parts {
		var found *paradox.Entry
		for _, e := range candidates {
			if guiName(e) == name {
				found = e
				break
			}
		}
		if found == nil {
			return nil, nil, -1
		}
		node = found
		if depth < len(parts)-1 {
			candidates = guiChildren(found)
		}
	}
	if node != nil {
		var find func(*paradox.Block) bool
		find = func(b *paradox.Block) bool {
			for i, e := range b.Entries {
				if e == node {
					parent, index = b, i
					return true
				}
				if e.Block != nil && find(e.Block) {
					return true
				}
			}
			return false
		}
		find(root)
	}
	return
}
func setGUIString(node *paradox.Entry, key, value string) {
	if node == nil || node.Block == nil {
		return
	}
	for _, e := range node.Block.Entries {
		if e.Key != nil && strings.EqualFold(*e.Key, key) {
			e.Block = nil
			e.Scalar = paradox.Scalar{Kind: paradox.ScalarStr, Str: value}
			e.Quoted = true
			return
		}
	}
	k, op := key, "="
	node.Block.Entries = append(node.Block.Entries, &paradox.Entry{Key: &k, Op: &op, Scalar: paradox.Scalar{Kind: paradox.ScalarStr, Str: value}, Quoted: true})
}

func setGUIImage(node *paradox.Entry, gfx string) {
	if node == nil || node.Block == nil {
		return
	}
	if setGUIImageKeys(node, gfx) {
		return
	}
	if node.Key != nil && strings.HasSuffix(strings.ToLower(*node.Key), "windowtype") {
		for _, e := range node.Block.Entries {
			if e.Key != nil && strings.EqualFold(*e.Key, "background") && e.Block != nil {
				if !setGUIImageKeys(e, gfx) {
					setGUIString(e, "quadTextureSprite", gfx)
				}
				return
			}
		}
		key, op, name := "background", "=", "name"
		background := &paradox.Entry{Key: &key, Op: &op, Block: &paradox.Block{}}
		background.Block.Entries = append(background.Block.Entries,
			&paradox.Entry{Key: &name, Op: &op, Scalar: paradox.Scalar{Kind: paradox.ScalarStr, Str: "Background"}, Quoted: true})
		setGUIString(background, "quadTextureSprite", gfx)
		node.Block.Entries = append(node.Block.Entries, background)
		return
	}
	setGUIString(node, "spriteType", gfx)
}

func setGUIImageKeys(node *paradox.Entry, gfx string) bool {
	found, textureFile := false, false
	for _, e := range node.Block.Entries {
		if e.Key == nil || e.Block != nil {
			continue
		}
		switch strings.ToLower(*e.Key) {
		case "quadtexturesprite", "spritetype", "sprite":
			e.Scalar = paradox.Scalar{Kind: paradox.ScalarStr, Str: gfx}
			e.Quoted = true
			found = true
		case "texturefile":
			textureFile = true
		}
	}
	if !found && textureFile {
		deleteGUIKey(node, "textureFile")
		setGUIString(node, "spriteType", gfx)
		found = true
	}
	return found
}

func guiRecoveriesWritable(recoveries []paradox.Recovery) bool {
	for _, recovery := range recoveries {
		if recovery.Code != paradox.RecoveryUnclosedBlock {
			return false
		}
	}
	return true
}

var engineSwappedGUIWindows = map[string]bool{"frontend_background": true, "load_screen": true}

func engineSwappedGUITarget(target string) bool {
	root, _, _ := strings.Cut(strings.TrimSpace(target), "/")
	return engineSwappedGUIWindows[root]
}

func hideGUINode(node *paradox.Entry) {
	if node == nil || node.Block == nil {
		return
	}
	setGUIPair(node, "position", "x", hiddenGUIOffset, "y", hiddenGUIOffset)
	setGUIWord(node, "hide", "yes")
}

const hiddenGUIOffset = -10000

func setGUIWord(node *paradox.Entry, key, word string) {
	for _, e := range node.Block.Entries {
		if e.Key != nil && strings.EqualFold(*e.Key, key) {
			e.Block = nil
			e.Scalar = paradox.Scalar{Kind: paradox.ScalarStr, Str: word}
			e.Quoted = false
			return
		}
	}
	k, op := key, "="
	node.Block.Entries = append(node.Block.Entries, &paradox.Entry{Key: &k, Op: &op, Scalar: paradox.Scalar{Kind: paradox.ScalarStr, Str: word}})
}

func deleteGUIKey(node *paradox.Entry, key string) {
	if node == nil || node.Block == nil {
		return
	}
	for i := len(node.Block.Entries) - 1; i >= 0; i-- {
		e := node.Block.Entries[i]
		if e.Key != nil && strings.EqualFold(*e.Key, key) {
			node.Block.Entries = append(node.Block.Entries[:i], node.Block.Entries[i+1:]...)
		}
	}
}
func setGUIPair(node *paradox.Entry, key, k1 string, v1 any, k2 string, v2 any) {
	if v1 == nil && v2 == nil {
		return
	}
	var target *paradox.Entry
	for _, e := range node.Block.Entries {
		if e.Key != nil && strings.EqualFold(*e.Key, key) && e.Block != nil {
			target = e
			break
		}
	}
	if target == nil {
		k, op := key, "="
		target = &paradox.Entry{Key: &k, Op: &op, Block: &paradox.Block{}}
		node.Block.Entries = append(node.Block.Entries, target)
	}
	setGUINumber(target, k1, v1)
	setGUINumber(target, k2, v2)
}
func setGUINumber(node *paradox.Entry, key string, value any) {
	if value == nil {
		return
	}
	text := fmt.Sprint(value)
	if number, ok := value.(float64); ok {

		text = strconv.FormatFloat(number, 'f', -1, 64)
	}
	kind := paradox.ScalarInt
	i, err := strconv.ParseInt(text, 10, 64)
	var scalar paradox.Scalar
	if err == nil {
		scalar = paradox.Scalar{Kind: kind, Int: newBigInt(i)}
	} else {
		scalar = paradox.Scalar{Kind: paradox.ScalarStr, Str: text}
	}
	k, op := key, "="
	for _, e := range node.Block.Entries {
		if e.Key != nil && strings.EqualFold(*e.Key, key) {
			e.Block = nil
			e.Scalar = scalar
			return
		}
	}
	node.Block.Entries = append(node.Block.Entries, &paradox.Entry{Key: &k, Op: &op, Scalar: scalar})
}
func newBigInt(v int64) *big.Int { return big.NewInt(v) }
func guiPayloadString(p map[string]any, key string) string {
	if v, ok := p[key].(string); ok {
		return v
	}
	return ""
}
func guiLangText(p map[string]any, lang string) string {
	if m, ok := p["lang_texts"].(map[string]any); ok {
		if v, ok := m[lang].(string); ok {
			return v
		}
	}
	return ""
}
func guiOverlayTextValue(p map[string]any) string {
	if key := strings.TrimSpace(guiPayloadString(p, "key")); key != "" {
		if isNew, supplied := p["is_new"].(bool); !supplied || isNew {
			return "GUISUB_" + key
		}
		return key
	}
	if text := guiPayloadString(p, "text"); text != "" {
		return text
	}
	if text := guiLangText(p, "en"); text != "" {
		return text
	}
	if texts, ok := p["lang_texts"].(map[string]any); ok {
		keys := make([]string, 0, len(texts))
		for key := range texts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if text, ok := texts[key].(string); ok {
				return text
			}
		}
	}
	return ""
}

func guiOverlayTextSupplied(p map[string]any) bool {
	if strings.TrimSpace(guiPayloadString(p, "key")) != "" {
		return true
	}
	if _, ok := p["text"].(string); ok {
		return true
	}
	if texts, ok := p["lang_texts"].(map[string]any); ok {
		for _, value := range texts {
			if _, ok := value.(string); ok {
				return true
			}
		}
	}
	return false
}

func guiOverlayLocalizationValue(p map[string]any, lang string) string {
	if text := guiLangText(p, lang); text != "" {
		return text
	}
	if text, ok := p["text"].(string); ok && text != "" {
		return text
	}
	if lang != "en" {
		if text := guiLangText(p, "en"); text != "" {
			return text
		}
	}
	if texts, ok := p["lang_texts"].(map[string]any); ok {
		keys := make([]string, 0, len(texts))
		for key := range texts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if text, ok := texts[key].(string); ok && text != "" {
				return text
			}
		}
	}
	return ""
}
func guiSpriteSlug(name string) string {
	stem := strings.TrimSuffix(strings.TrimSuffix(name, ".png"), ".PNG")
	var b strings.Builder
	for _, r := range stem {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		return "sprite"
	}
	return s
}
func dedupeSprites(in []emitmisc.SpritePair) []emitmisc.SpritePair {
	seen := map[string]bool{}
	out := make([]emitmisc.SpritePair, 0, len(in))
	for _, x := range in {
		if !seen[x.Name] {
			seen[x.Name] = true
			out = append(out, x)
		}
	}
	return out
}

func escapeYML(s string) string {
	return locout.YmlEscapeValue(s)
}

func guiEntryFromJSON(raw map[string]any) *paradox.Entry {
	typeName, _ := raw["type"].(string)
	if strings.TrimSpace(typeName) == "" {
		typeName = "containerWindowType"
	}
	k, op := typeName, "="
	entry := &paradox.Entry{Key: &k, Op: &op, Block: &paradox.Block{}}
	for _, key := range []string{"name", "text", "sprite", "orientation", "origo"} {
		if value, ok := raw[key].(string); ok && value != "" {
			setGUIString(entry, key, value)
		}
	}
	if position, ok := raw["position"].(map[string]any); ok {
		setGUIPair(entry, "position", "x", position["x"], "y", position["y"])
	}
	if size, ok := raw["size"].(map[string]any); ok {
		setGUIPair(entry, "size", "width", size["width"], "height", size["height"])
	}
	if children, ok := raw["children"].([]any); ok {
		for _, child := range children {
			if obj, ok := child.(map[string]any); ok {
				entry.Block.Entries = append(entry.Block.Entries, guiEntryFromJSON(obj))
			}
		}
	}
	return entry
}
