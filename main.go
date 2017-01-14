package main

import (
	"context"
	"log"
)

func main() {
	pad, err := OpenLaunchpad(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer pad.Close()

	pad.Reset()
	pad.Main()

	if err := pad.Wait(); err != nil {
		log.Fatal(err)
	}
}
