package main

import (
	"errors"
	"os/exec"
	"runtime"
)

var errNoWindow = errors.New("no program window on this system")

type window interface {
	run()

	close()

	show()
}

func openDefaultBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
