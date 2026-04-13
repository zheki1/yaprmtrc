package main

import (
	"log"
	"os"
)

func main() {
	os.Exit(1) // want `call to os\.Exit in the main package is not allowed`
}

func helper() {
	log.Fatal("bye")   // want `call to log\.Fatal in the main package is not allowed`
	log.Fatalf("bye")  // want `call to log\.Fatalf in the main package is not allowed`
	log.Fatalln("bye") // want `call to log\.Fatalln in the main package is not allowed`
	os.Exit(2)         // want `call to os\.Exit in the main package is not allowed`
	panic("oops")      // want `call to panic in the main package is not allowed`
}
