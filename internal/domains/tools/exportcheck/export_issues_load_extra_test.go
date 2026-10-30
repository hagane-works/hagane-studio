package exportcheck

import (
	"errors"
)

var ErrExportGateIncomplete = errors.New(
	"export gate assembly incomplete: localisation-reference scan and " +
		"content-language closure groups are not ported yet")
