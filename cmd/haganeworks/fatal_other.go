//go:build !windows

package main

import "log"

func reportError(err error) {
	log.Print(err)
}
