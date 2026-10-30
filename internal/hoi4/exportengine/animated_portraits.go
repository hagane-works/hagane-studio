package exportengine

import (
	"fmt"
	"sort"
	"strings"

	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

func AnimatedPortraitStem(gfxName string) string {
	stem := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(gfxName), "GFX_"))
	var b strings.Builder
	for _, r := range stem {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "animated_portrait"
	}
	return b.String()
}

func AnimatedPortraitLargePath(gfxName string) string {
	return "gfx/leaders/asset_lab/" + AnimatedPortraitStem(gfxName) + "_animated.dds"
}

func AnimatedPortraitSmallPath(gfxName string) string {
	return "gfx/leaders/asset_lab/" + AnimatedPortraitStem(gfxName) + "_small.dds"
}

func AnimatedPortraitGFXPath(gfxName string) string {
	return "interface/" + AnimatedPortraitStem(gfxName) + "_portrait.gfx"
}

func BuildAnimatedPortraitGFX(gfxName, smallGFXName string, frameCount, fps int) string {
	return fmt.Sprintf(`spriteTypes = {
	frameAnimatedSpriteType = {
		name = "%s"
		texturefile = "%s"
		noOfFrames = %d
		animation_rate_fps = %d
		looping = yes
		play_on_show = yes
	}
	spriteType = {
		name = "%s"
		texturefile = "%s"
	}
}
`, gfxName, AnimatedPortraitLargePath(gfxName), frameCount, fps, smallGFXName, AnimatedPortraitSmallPath(gfxName))
}

func nativeAnimatedPortraits(snapshot Snapshot, modDir string, includeImported bool, resolve BlobResolver, sink *imageSink) ([]Entry, error) {
	assets, err := projectOBSAssets(snapshot)
	if err != nil {
		return nil, err
	}
	var names []string
	for key := range assets {
		if strings.HasPrefix(key, "animated_portrait_gfx\x00") {
			names = append(names, strings.TrimPrefix(key, "animated_portrait_gfx\x00"))
		}
	}
	if len(names) == 0 {
		return nil, nil
	}
	referenced := map[string]bool{}
	for _, raw := range snapshot.Tables["characters"] {
		entity, derr := emitmisc.DecodeEntityJSON(raw)
		if derr != nil {
			return nil, fmt.Errorf("decode character: %w", derr)
		}
		if !includeImported && entity.Str("source") == "imported" {
			continue
		}
		if ref := strings.TrimSpace(entity.Str("large_portrait")); strings.HasPrefix(ref, "GFX_") {
			referenced[ref] = true
		}
	}
	sort.Strings(names)
	var entries []Entry
	for _, name := range names {
		if !referenced[name] {
			continue
		}
		gfxText, ok, err := ResolveBlobTolerant(resolve, assets["animated_portrait_gfx\x00"+name], 1<<20, "animated portrait gfx "+name)
		if err != nil {
			return nil, err
		}
		large, lok, err := ResolveBlobTolerant(resolve, assets["animated_portrait\x00"+name], 32<<20, "animated portrait sheet "+name)
		if err != nil {
			return nil, err
		}
		small, sok, err := ResolveBlobTolerant(resolve, assets["animated_portrait_small\x00"+name], 8<<20, "animated portrait small "+name)
		if err != nil {
			return nil, err
		}
		if !ok || !lok || !sok || len(gfxText) == 0 || len(large) == 0 || len(small) == 0 {

			sink.skipped("animated portrait " + name)
			continue
		}
		entries = append(entries,
			Entry{Path: modDir + "/" + AnimatedPortraitGFXPath(name), Bytes: gfxText, Origin: "generated", SourceEntity: "animated_portrait_gfx"},
			Entry{Path: modDir + "/" + AnimatedPortraitLargePath(name), Bytes: large, Origin: "private_asset", SourceEntity: "animated_portrait"},
			Entry{Path: modDir + "/" + AnimatedPortraitSmallPath(name), Bytes: small, Origin: "private_asset", SourceEntity: "animated_portrait"},
		)
	}
	return entries, nil
}

func passthroughPortraitRef(ref string, hasBytes bool) string {
	ref = strings.TrimSpace(ref)
	if hasBytes || ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "GFX_") || strings.HasPrefix(ref, "gfx/") {
		return ref
	}
	return ""
}
