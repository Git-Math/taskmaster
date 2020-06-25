package main

import (
	"fmt"
	"os"
	"taskmaster/parse_yaml"
	"taskmaster/term"
)

func usage() {
	fmt.Println("usage:", os.Args[0], "[-h] yamlfile")
	fmt.Println()
	fmt.Println("positional argument:")
	fmt.Println("  yamlfile:                  yaml config for programs")
	fmt.Println()
	fmt.Println("  -h, --help                 show this help message and exit")
}

func main() {

	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	} else if os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		os.Exit(0)
	}

	program_map := parse_yaml.parse_yaml(os.Args[1])

	term.Init()

	for {
		text := term.ReadLine()
		fmt.Println(text)
	}
}
