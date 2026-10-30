package guirender

type ProjectRawTexture struct {
	Path   string
	Width  int
	Height int

	Version string

	ObjectKey string
}

func (idx *Index) SetProjectRawTextures(textures []ProjectRawTexture) {
	if idx == nil {
		return
	}
	if len(textures) == 0 {
		idx.projectRawTextures = nil
		return
	}
	idx.projectRawTextures = make(map[string]ProjectRawTexture, len(textures))
	for _, texture := range textures {
		if key := foldTexturePath(texture.Path); key != "" && texture.Width > 0 && texture.Height > 0 {
			idx.projectRawTextures[key] = texture
		}
	}
}

func (idx *Index) projectRawTexture(value string) (ProjectRawTexture, bool) {
	variants := textureVariants(value)
	for layer := idx; layer != nil; layer = layer.fallback {
		if len(layer.projectRawTextures) == 0 {
			continue
		}
		for _, variant := range variants {
			if texture, ok := layer.projectRawTextures[variant]; ok {
				return texture, true
			}
		}
	}
	return ProjectRawTexture{}, false
}

func (idx *Index) ProjectRawSpriteTexture(name string) (ProjectRawTexture, bool) {
	meta, found := idx.spriteRecordFor(name)
	if !found {
		return ProjectRawTexture{}, false
	}
	for _, candidate := range meta.TextureFiles {
		if texture, ok := idx.projectRawTexture(candidate); ok {
			return texture, true
		}
	}
	return ProjectRawTexture{}, false
}
