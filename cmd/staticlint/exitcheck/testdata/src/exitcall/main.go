package main

import (
	"log"
	"os"
)

func main() {
	os.Exit(1)
}

func helper() {
	log.Fatal("bye")
	log.Fatalf("bye")
	log.Fatalln("bye")
	os.Exit(2)
	panic("oops")
}
