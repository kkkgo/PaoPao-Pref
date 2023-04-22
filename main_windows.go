package main

import (
	"runtime"

	"golang.org/x/sys/windows"
)

func init() {
	if runtime.GOOS == "windows" {
		var originalMode uint32
		stdout := windows.Handle(windows.Stdout)
		windows.GetConsoleMode(stdout, &originalMode)
		windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
