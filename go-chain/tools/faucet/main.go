package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintf(os.Stderr, "Usage: faucet [server]\n")
	fmt.Fprintf(os.Stderr, "  server  start the faucet server\n")
	os.Exit(1)
}
