package domains

import (
	"io/fs"

	"haganeworks/internal/apiutil"
)

type Deps struct {
	Open apiutil.Opener

	Game func() (fs.FS, error)
}
