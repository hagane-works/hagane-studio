package toolapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// An exported ZIP is copied to the file as it arrives, up to the cap.
func TestStreamToCopiesUpToTheCap(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer upstream.Close()
	proxy := &RESTProxy{BaseURL: upstream.URL, Client: upstream.Client()}

	var out bytes.Buffer
	if n, err := proxy.streamTo(context.Background(), "/zip", &out, 10); err != nil || n != 10 || out.String() != "0123456789" {
		t.Fatalf("at the cap: n=%d err=%v out=%q", n, err, out.String())
	}
	out.Reset()
	if _, err := proxy.streamTo(context.Background(), "/zip", &out, 9); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("over the cap: err=%v", err)
	}
}
