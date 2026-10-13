package main

import (
	"os"
	system "os"
)

func main() {
	os.Exit(1)     // want "прямой вызов os.Exit запрещён в функции main"
	system.Exit(2) // want "прямой вызов os.Exit запрещён в функции main"
}

func shutdown() {
	os.Exit(1) // Допустимо: вызов не внутри main
}
