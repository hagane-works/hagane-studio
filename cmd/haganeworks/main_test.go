package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListen(t *testing.T) {
	portOf := func(listener net.Listener) int { return listener.Addr().(*net.TCPAddr).Port }
	freePort := func() int {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		return portOf(listener)
	}
	listenOn := func(port int, anyFree bool) (net.Listener, error) {
		listener, err := listen(port, anyFree)
		if listener != nil {
			t.Cleanup(func() { listener.Close() })
		}
		return listener, err
	}

	port := freePort()
	if got, err := listenOn(port, false); err != nil || portOf(got) != port {
		t.Errorf("free port %d: %v %v", port, got, err)
	}

	running := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Hagane-Works", "1")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer running.Close()
	for _, anyFree := range []bool{true, false} {
		if got, err := listenOn(portOf(running.Listener), anyFree); got != nil || err != nil {
			t.Errorf("a copy running there (anyFree %v): %v %v", anyFree, got, err)
		}
	}

	other := httptest.NewServer(http.NotFoundHandler())
	defer other.Close()
	if got, err := listenOn(portOf(other.Listener), true); err != nil || portOf(got) == portOf(other.Listener) {
		t.Errorf("another program there, any free port: %v %v", got, err)
	}
	if got, err := listenOn(portOf(other.Listener), false); got != nil || err == nil {
		t.Errorf("another program there, this port only: %v %v", got, err)
	}
}
