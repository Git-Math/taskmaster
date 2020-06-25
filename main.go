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
	fmt.Println("status:", program_map)
}

func start(program string) {
	fmt.Println("start:", program)
}

func stop(program string) {
	fmt.Println("stop:", program)
}

func restart(program string) {
	fmt.Println("restart:", program)
}

func reload_config() {
	fmt.Println("reload_config")
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
			fmt.Println("start command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			start(arg)
		}
	case "stop":
		if len(args) == 0 {
			fmt.Println("stop command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			stop(arg)
		}
	case "restart":
		if len(args) == 0 {
			fmt.Println("restart command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			restart(arg)
		}
	case "reload_config":
		reload_config()
	case "exit":
		exit()
	default:
		fmt.Println("command not found:", cmd)
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

	program_map := parse_yaml.ParseYaml(os.Args[1])

	term.Init()

	for {
		text := term.ReadLine()
		call_func(text, program_map)
	}
}
