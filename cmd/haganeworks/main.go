package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"haganeworks/internal/api"
	"haganeworks/internal/export"
	"haganeworks/internal/server"
	"haganeworks/internal/settings"
	"haganeworks/internal/toolapi"
	"haganeworks/web"
)

var version = "dev"

func main() {
	port := flag.Int("port", 0, fmt.Sprintf("local port to listen on (default: the settings' port, else %d; a free one when that is taken)", settings.DefaultPort))
	noWindow := flag.Bool("no-window", false, "start the server without opening a window")
	showVersion := flag.Bool("version", false, "print the version and exit")
	configPath := flag.String("config", "", "settings file to use (default: the user config folder)")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if err := run(*port, *noWindow, *configPath); err != nil {
		reportError(err)
		os.Exit(1)
	}
}

func run(port int, noWindow bool, settingsPath string) error {
	if settingsPath == "" {
		var err error
		if settingsPath, err = settings.DefaultPath(); err != nil {
			return err
		}
	}
	store, err := settings.Open(settingsPath)
	if err != nil {
		return fmt.Errorf("read settings %s: %w", settingsPath, err)
	}

	want, anyFree := port, port == 0
	if anyFree {
		want = store.Get().ListenPort()
	}
	listener, err := listen(want, anyFree)
	if err != nil {
		return err
	}
	if listener == nil {

		log.Printf("Hagane Works is already running at http://127.0.0.1:%d/", want)
		if noWindow {
			return nil
		}
		return showRunningCopy(want)
	}
	host := listener.Addr().String()

	toolsToken, err := store.ToolsToken()
	if err != nil {
		log.Printf("tools token: %v; the tool API stays closed", err)
	}
	srv, err := server.New(host, web.Dist(), toolsToken)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	quit := make(chan struct{}, 1)
	exports, err := export.NewJobs()
	if err != nil {
		return err
	}
	defer exports.Close()
	api.Register(srv.Mux, api.Deps{
		Version:  version,
		Settings: store,
		Exports:  exports,
		Quit:     func() { quit <- struct{}{} },
		Port:     listener.Addr().(*net.TCPAddr).Port,
	})
	toolapi.Register(srv.Tools, srv.Mux)

	httpServer := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()

	url := "http://" + host + "/"
	log.Printf("Hagane Works %s at %s", version, url)
	var win window
	if !noWindow {
		if win, err = newWindow(url, filepath.Join(filepath.Dir(settingsPath), "window")); err != nil {

			if errors.Is(err, errNoWindow) {
				log.Printf("program window: %v; opening the default browser", err)
			} else {
				reportError(fmt.Errorf("Hagane Works could not open its window: %w\n\nIt opens in your default browser instead.", err))
			}
			if err := openDefaultBrowser(url); err != nil {
				log.Printf("open browser: %v (open %s yourself)", err, url)
			}
		}
	}
	if win != nil {

		srv.Sessionless.HandleFunc("POST /api/app/show", func(w http.ResponseWriter, r *http.Request) {
			win.show()
			w.WriteHeader(http.StatusNoContent)
		})
	}

	windowClosed := make(chan struct{})
	ended := make(chan error, 1)
	go func() {
		var err error
		select {
		case <-ctx.Done():
		case <-quit:
		case <-windowClosed:
		case err = <-serveErr:
		}
		if win != nil {
			win.close()
		}
		ended <- err
	}()
	if win != nil {
		win.run()
		close(windowClosed)
	}
	if err := <-ended; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdown)
}

func listen(port int, anyFree bool) (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err == nil {
		return listener, nil
	}
	if runningCopy(port) {
		return nil, nil
	}
	if !anyFree {
		return nil, err
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func runningCopy(port int) bool {
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/app/ping", port))
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.Header.Get("X-Hagane-Works") != ""
}

func showRunningCopy(port int) error {
	allowForeground()
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/api/app/show", port), "", nil)
	if err == nil {
		res.Body.Close()
		if res.StatusCode == http.StatusNoContent {
			return nil
		}
	}
	return openDefaultBrowser(fmt.Sprintf("http://127.0.0.1:%d/", port))
}
