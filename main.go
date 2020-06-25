package main

import (
	"fmt"
	"taskmaster/term"
)

func main() {

	term.Init()

	for {
		text := term.ReadLine()
		fmt.Println(text)
	}
}
