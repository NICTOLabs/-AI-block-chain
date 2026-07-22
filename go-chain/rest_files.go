// +build ignore

package main

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	pkg := build.Default
	matches, _ := filepath.Glob("*.go")
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") || strings.HasSuffix(m, ".s") {
			continue
		}
		fmt.Println(m)
	}
}
