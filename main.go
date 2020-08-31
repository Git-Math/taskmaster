package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
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

func start(name string, cfg parse_yaml.Program) {
	tasks.StartProgram(name, cfg)
}

func stop(program_name string, cfg parse_yaml.Program) {
	var wg sync.WaitGroup

	daemons := tasks.DaemonRetrieve(program_name)
	wg.Add(len(daemons))

	for _, daemon := range daemons {
		if daemon != nil && daemon.IsRunning() {
			go func() {
				defer wg.Done()
				tasks.StopProgram(cfg, daemon)
			}()
		}
	}

	wg.Wait()
}

func restart(program_name string, cfg parse_yaml.Program) {
	stop(program_name, cfg)
	start(program_name, cfg)
}

func RemoveProgram(program_name string, cfg parse_yaml.Program) {
	stop(program_name, cfg)
}

func reload_config(program_map parse_yaml.ProgramMap, cfg_yaml string) parse_yaml.ProgramMap {
	new_program_map, err := parse_yaml.ParseYaml(cfg_yaml)
	if err != nil {
		fmt.Print(err)
		return program_map
	}

	for key, cfg := range program_map {
		if _, key_exist := new_program_map[key]; !key_exist {
			RemoveProgram(key, cfg)
		}
	}

	return new_program_map
}

func exit(program_map parse_yaml.ProgramMap) {
	var wg sync.WaitGroup

	wg.Add(len(program_map))

	for key, cfg := range program_map {
		go func() {
			defer wg.Done()
			stop(key, cfg)
		}()
	}

	wg.Wait()
	os.Exit(0)
}

func call_func(text string, program_map parse_yaml.ProgramMap, cfg_yaml string) {
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
		program_map = reload_config(program_map, cfg_yaml)
	case "exit":
		exit(program_map)
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

	cfg_yaml := os.Args[1]

	program_map, err := parse_yaml.ParseYaml(cfg_yaml)
	if err != nil {
		log.Fatal(err)
	}

	for name, program := range program_map {
		tasks.Add(name, program)
		if program.Autostart {
			tasks.StartProgram(name, program)
		}
	}

	go master.Watch(program_map)

	term.Init()

	for {
		text := term.ReadLine()
		call_func(text, program_map, cfg_yaml)
	}
}
