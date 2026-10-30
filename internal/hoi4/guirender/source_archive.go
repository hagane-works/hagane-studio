package guirender

func (idx *Index) SetSourceArchiveRevision(key, revision string) {
	if idx == nil {
		return
	}
	idx.sourceArchiveKey = key
	idx.sourceArchiveRevision = revision
}

func (idx *Index) SourceArchiveRevision() (key, revision string, ok bool) {
	for layer := idx; layer != nil; layer = layer.fallback {
		if layer.sourceArchiveKey != "" {
			return layer.sourceArchiveKey, layer.sourceArchiveRevision, true
		}
	}
	return "", "", false
}
