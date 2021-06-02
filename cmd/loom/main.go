package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/pgavlin/loom"
)

func main() {
	var r io.Reader

	switch len(os.Args) {
	case 1:
		r = os.Stdin
	case 2:
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		r = f
	case 3:
		fmt.Fprintf(os.Stderr, "usage: %s [path to file]\n", os.Args[0])
		os.Exit(-1)
	}

	x, err := loom.Parse(r)
	if err != nil {
		log.Fatalf("error parsing input: %v", err)
	}

	loom.Encode(os.Stdout, loom.NewEnv().Eval(x))
	fmt.Printf("\n")
}
