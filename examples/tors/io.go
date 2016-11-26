// +build !windows

package main

import (
	"io"
	"os"
)

func getStdin() io.Reader {
	return os.Stdin
}

func getStdout() io.Writer {
	return os.Stdout
}
