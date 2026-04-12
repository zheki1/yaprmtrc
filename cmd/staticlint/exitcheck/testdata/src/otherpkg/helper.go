package helper

import "os"

func Exit() {
	os.Exit(1) // allowed: not main func of main package
}
