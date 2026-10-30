package main

import (
	"log"
	"syscall"
	"unsafe"
)

var messageBox = user32.NewProc("MessageBoxW")

func reportError(err error) {
	log.Print(err)
	title, _ := syscall.UTF16PtrFromString("Hagane Works")
	text, _ := syscall.UTF16PtrFromString(err.Error())
	const iconError = 0x10
	_, _, _ = messageBox.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), iconError)
}
