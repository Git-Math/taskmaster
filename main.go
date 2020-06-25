package main

import (
	"fmt"
	"os"
	"strings"
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

func status(program_map parse_yaml.ProgramMap) {
	fmt.Println("status:", ProgramMap)
}

func start(program string) {
	fmt.Println("start:", porgram)
}

func stop(program string) {
	fmt.Println("stop:", program)
}

func restart(program string) {
	fmt.Println("restart:", program)
}

func exit() {
	fmt.Println("exit")
}

func call_func(text string, program_map parse_yaml.ProgramMap) {
	text_list := strings.Fields(text)

	if len(text_list) == 0 {
		return
	}

	cmd := text_list[0]
	args := text_list[1:]
	switch cmd {
	case "status":
		status(program_map)
	case "start":
		if len(args) == 0 {
			fmt.Printf("start command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			start(arg)
		}
	case "stop":
		if len(args) == 0 {
			fmt.Printf("stop command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			stop(arg)
		}
	case "restart":
		if len(args) == 0 {
			fmt.Printf("restart command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			restart(arg)
		}
	case "exit":
		exit()
	default:
		fmt.Println("Command not found:", cmd)
	}
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
		call_func(text, program_map)
	}
}
