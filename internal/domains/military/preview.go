package military

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"haganeworks/internal/apiutil"
	emitmisc "haganeworks/internal/hoi4/emit/misc"
)

func writePreview(w http.ResponseWriter, entity any, generate func(*emitmisc.NS) string, op string) {
	body, err := json.Marshal(entity)
	if err == nil {
		var node *emitmisc.NS
		if node, err = emitmisc.DecodeEntityJSON(body); err == nil {
			writeJSON(w, http.StatusOK, map[string]string{"script": generate(node)})
			return
		}
	}
	log.Printf("%s: %v", op, err)
	detail(w, http.StatusInternalServerError, "Internal error")
}

func previewFail(w http.ResponseWriter, err error, op string) {
	var notFound apiutil.NotFoundError
	if errors.As(err, &notFound) {
		apiutil.Fail(w, err)
		return
	}
	log.Printf("%s: %v", op, err)
	detail(w, http.StatusInternalServerError, "Internal error")
}
