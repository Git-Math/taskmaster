package main

import (
	"fmt"
	l "log"
	"os"
	"strings"
	"sync"
	"taskmaster/log"
	"taskmaster/master"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"taskmaster/term"
	"time"
)

func usage() {
	fmt.Println("usage:", os.Args[0], "[-h|-v] yamlfile")
	fmt.Println()
	fmt.Println("positional argument:")
	fmt.Println("  yamlfile:                  yaml config for programs")
	fmt.Println()
	fmt.Println("  -v                         verbose in stdout")
	fmt.Println("  -h, --help                 show this help message and exit")
}

func status(program_map parse_yaml.ProgramMap) {
	for name, daemons := range tasks.Daemons {
		fmt.Println("Program name:", name)
		fmt.Println("Cmd:         ", program_map[name].Cmd)
		for i, daemon := range daemons {
			fmt.Printf("    Process %d:\n", i)
			fmt.Println("        No restart:   ", daemon.NoRestart)
			fmt.Println("        Start time:   ", time.Unix(daemon.StartTime/tasks.SecondToMillisecond, 0))
			fmt.Println("        Start retries:", daemon.StartRetries)
			fmt.Println("        Running:      ", daemon.Running)
			fmt.Println("        Exit code:    ", daemon.ExitCode)
		}
	}
}

func start(name string, cfg parse_yaml.Program) {
	tasks.StartProgram(name, cfg)
}

func stop(name string, cfg parse_yaml.Program) {
	tasks.StopProgram(name, cfg)
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
	var cfg_yaml string
	if len(os.Args) == 3 {
		if os.Args[1] == "-v" {
			log.Debug = l.New(os.Stdout, "DEBUG: ", l.Ldate|l.Ltime|l.Lshortfile)
			cfg_yaml = os.Args[2]
		} else {
			usage()
			os.Exit(1)
		}
	} else if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	} else if os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		os.Exit(0)
	} else {
		file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			l.Fatal(err)
		}
		log.Debug = l.New(file, "DEBUG: ", l.Ldate|l.Ltime|l.Lshortfile)
		cfg_yaml = os.Args[1]
	}

	program_map, err := parse_yaml.ParseYaml(cfg_yaml)
	if err != nil {
		l.Fatal(err)
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
