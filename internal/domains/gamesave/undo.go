package gamesave

import (
	"errors"
	"net/http"

	"haganeworks/internal/domains/gameview"
	"haganeworks/internal/project"
)

func undoRawFiles(st *store, steps []rawFileStep) error {
	if len(steps) == 0 {
		return saveError{422, "nothing to undo"}
	}
	current := make([]*gameview.RawFile, len(steps))
	for i, step := range steps {
		file, err := st.raw.get(step.ID)
		switch {
		case errors.Is(err, gameview.ErrRawFileMissing):
			if step.Before != nil {
				return saveError{409, step.File + " 已经不在项目里了,撤销不了"}
			}
			continue
		case err != nil:
			return err
		}
		if textSHA256(file.Content) != step.AfterSHA256 {
			return saveError{409, step.File + " 在这次保存之后又改过(比如在工作台里),撤销会丢掉那些改动,所以没撤销"}
		}
		if step.Before != nil {
			if err := gateText(file.DestPath, *step.Before); err != nil {
				return err
			}
		}
		current[i] = &file
	}
	for i := len(steps) - 1; i >= 0; i-- {
		file, step := current[i], steps[i]
		if file == nil {
			continue
		}
		if step.Before == nil {
			if err := st.raw.delete(step.ID); err != nil {
				return err
			}
			continue
		}
		note := step.BeforeNote
		if _, err := st.raw.update(step.ID, gameview.RawFileIntent{
			RawKind: file.RawKind, DestPath: file.DestPath, Content: *step.Before, GenerateReplacePath: file.GenerateReplacePath,
			TargetLayer: file.TargetLayer, TargetSHA256: file.TargetSHA256,
		}, &note); err != nil {
			return err
		}
	}
	return nil
}

func (h *handlers) undoFiles(p *project.Project, w http.ResponseWriter, r *http.Request) {
	var in struct {
		Files []rawFileStep `json:"files"`
	}
	if !decodeBody(w, r, 32<<20, &in, "invalid undo payload") {
		return
	}
	_, err := write(p, []string{rawFilesTable}, func(st *store) (struct{}, error) {
		return struct{}{}, undoRawFiles(st, in.Files)
	})
	h.finish(w, p, map[string]any{"undone": len(in.Files)}, err, "undo", true)
}
