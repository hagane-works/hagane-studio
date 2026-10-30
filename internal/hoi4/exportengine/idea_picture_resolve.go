package exportengine

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"haganeworks/internal/hoi4/exportmsg"
	"haganeworks/internal/hoi4/vanilladata"
)

const IdeaPictureDroppedCode = "idea_picture_dropped"

const ideaPictureSuggestionLimit = 3

type IdeaPictureDrop struct {
	RowID       int64
	IdeaID      string
	Picture     string
	Stripped    string
	Suggestions []string
}

func resolveIdeaPictureRefs(ideas []nativeRow, ideaRefs map[int64]string, isSubmod bool, ownSprites map[string]bool, sink *imageSink) {
	for _, row := range ideas {
		if row.misc == nil {
			continue
		}
		id := row.misc.Int("id", 0)
		if ideaRefs[id] != "" {
			continue
		}
		raw := strings.TrimSpace(row.misc.Str("picture"))
		if raw == "" {
			continue
		}
		stripped := strings.TrimPrefix(raw, "GFX_idea_")
		if stripped == "" {
			continue
		}
		if vanilladata.IsVanillaIdeaPicture(stripped) || isSubmod || ownSprites[strings.ToLower("GFX_idea_"+stripped)] {
			ideaRefs[id] = stripped
			continue
		}
		drop := IdeaPictureDrop{
			RowID:       id,
			IdeaID:      row.misc.Str("idea_id"),
			Picture:     raw,
			Stripped:    stripped,
			Suggestions: vanilladata.SuggestIdeaPictures(stripped, ideaPictureSuggestionLimit),
		}
		sink.note(ideaPictureDropWarning(drop, sink.lang()))
		sink.ideaPictureDropped(drop)
	}
}

func (s *imageSink) ideaPictureDropped(drop IdeaPictureDrop) {
	if s == nil {
		return
	}
	s.ideaPictureDrops = append(s.ideaPictureDrops, drop)
}

func ideaPictureDropWarning(drop IdeaPictureDrop, msg exportmsg.Printer) string {
	warning := msg.Text("idea_picture.warning", drop.args())

	if len(drop.Suggestions) > 0 {
		warning += " " + msg.Text("idea_picture.did_you_mean", drop.args())
	}
	return warning
}

func (d IdeaPictureDrop) args() exportmsg.Args {
	return exportmsg.Args{"idea": d.IdeaID, "picture": d.Picture, "sprite": d.Stripped, "names": strings.Join(d.Suggestions, ", ")}
}

func (d IdeaPictureDrop) Message(msg exportmsg.Printer) string {
	return msg.Text("idea_picture.message", d.args())
}

func (d IdeaPictureDrop) PlayerImpact(msg exportmsg.Printer) string {
	return msg.Text("idea_picture.impact", nil)
}

func (d IdeaPictureDrop) EditRoute(msg exportmsg.Printer) string {
	if len(d.Suggestions) > 0 {
		return msg.Text("idea_picture.edit_with_names", d.args())
	}
	return msg.Text("idea_picture.edit", d.args())
}

var gfxSpriteName = regexp.MustCompile(`(?i)\bname\s*=\s*(?:"([^"]+)"|([A-Za-z0-9_.\-]+))`)

func projectSpriteNames(snapshot Snapshot, includeImported bool) map[string]bool {
	out := map[string]bool{}
	for _, raw := range snapshot.Tables["project_raw_files"] {
		var row struct {
			DestPath string `json:"dest_path"`
			Content  string `json:"content"`
			IsBinary bool   `json:"is_binary"`
			RawKind  string `json:"raw_kind"`
			Source   string `json:"source"`
		}
		if json.Unmarshal(raw, &row) != nil || row.IsBinary || row.RawKind == "empty_shadow" || row.RawKind == "tombstone" {
			continue
		}
		if !includeImported && row.Source == "imported" {
			continue
		}
		dest := strings.ToLower(strings.Trim(strings.ReplaceAll(row.DestPath, "\\", "/"), "/"))
		if !strings.HasPrefix(dest, "interface/") || path.Ext(dest) != ".gfx" {
			continue
		}
		for _, match := range gfxSpriteName.FindAllStringSubmatch(row.Content, -1) {
			name := match[1]
			if name == "" {
				name = match[2]
			}
			if name = strings.TrimSpace(name); name != "" {
				out[strings.ToLower(name)] = true
			}
		}
	}
	return out
}
