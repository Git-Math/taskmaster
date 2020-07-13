package main

import (
	"fmt"
	"os"
	"strings"
	"taskmaster/master"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
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

func start(program_name string, cfg parse_yaml.Program) {
	for _, daemon := range tasks.Daemons {
		if daemon.Name == program_name && !daemon.Running {
			tasks.StartProgram(cfg, daemon)
		}
	}
}

func stop(program_name string, cfg parse_yaml.Program) {
	for _, daemon := range tasks.Daemons {
		if daemon.Name == program_name && daemon.Running {
			tasks.StopProgram(cfg, daemon)
		}
	}
}

func restart(program_name string, cfg parse_yaml.Program) {
	stop(program_name, cfg)
	start(program_name, cfg)
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
			if _, key_exist := program_map[arg]; !key_exist {
				fmt.Println("start: program [", arg, "] does not exist")
			} else {
				start(arg, program_map[arg])
			}
		}
	case "stop":
		if len(args) == 0 {
			fmt.Println("stop command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			if _, key_exist := program_map[arg]; !key_exist {
				fmt.Println("stop: program [", arg, "] does not exist")
			} else {
				stop(arg, program_map[arg])
			}
		}
	case "restart":
		if len(args) == 0 {
			fmt.Println("restart command needs at least one program name as argument")
			return
		}
		for _, arg := range args {
			if _, key_exist := program_map[arg]; !key_exist {
				fmt.Println("restart: program [", arg, "] does not exist")
			} else {
				restart(arg, program_map[arg])
			}
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

	tasks.Execute(program_map)
	go master.Watch(program_map)

	term.Init()

	for {
		text := term.ReadLine()
		call_func(text, program_map)
	}
}
