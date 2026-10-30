package guirender

type SpriteOverride struct {
	Width, Height int

	URL string
}

type SpriteTextureGeometry struct {
	Width, Height int
	Frames        int
	Texture       string
}

func TextureImageSize(name string, body []byte) (int, int, bool) {
	return imageSize(name, body)
}

func (idx *Index) SetSpriteOverrides(overrides map[string]SpriteOverride) {
	idx.spriteOverrides = overrides
}

func (idx *Index) spriteOverride(name string) (SpriteOverride, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if override, ok := layer.spriteOverrides[name]; ok {
			return override, true
		}
	}
	return SpriteOverride{}, false
}

func (idx *Index) SpriteTextureGeometry(name string) (SpriteTextureGeometry, bool) {
	meta, declared := idx.spriteRecordFor(name)
	if declared {
		width, height := 0, 0
		texture := ""
		for _, candidate := range meta.TextureFiles {
			if texture == "" {
				texture = candidate
			}
			if w, h, _, ok := idx.textureSize(candidate); ok {
				width, height, texture = w, h, candidate
				break
			}
		}
		if (width <= 0 || height <= 0) && meta.NaturalSize != nil {
			width = toInt(meta.NaturalSize["x"], 0)
			height = toInt(meta.NaturalSize["y"], 0)
		}
		hintW, hintH := spriteSizeHint(name, meta.TextureFiles)
		if width <= 0 {
			width = hintW * max(1, meta.Frames)
		}
		if height <= 0 {
			height = hintH
		}
		frames := max(1, meta.Frames)
		if frames == 1 && hintW > 0 && width >= hintW*2 && width%hintW == 0 && width/hintW <= 16 && (hintH == 0 || height == 0 || abs(height-hintH) <= 4) {
			frames = width / hintW
		}
		return SpriteTextureGeometry{Width: width, Height: height, Frames: frames, Texture: texture}, true
	}
	if file, ok := idx.ScreenPictureFile(name); ok {
		width, height, _, resolved := idx.textureSize(file)
		if !resolved {
			return SpriteTextureGeometry{Frames: 1, Texture: file}, true
		}
		return SpriteTextureGeometry{Width: width, Height: height, Frames: 1, Texture: file}, true
	}
	return SpriteTextureGeometry{}, false
}

func (idx *Index) SpriteSource(name string) (string, []byte, bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if record, ok := layer.sprites[name]; ok && record.File != "" {
			body, ok := idx.Source(record.File)
			return record.File, body, ok
		}
	}
	return "", nil, false
}
