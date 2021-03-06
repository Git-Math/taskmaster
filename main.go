package main

import (
	"fmt"
	l "log"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"strings"
	"syscall"
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
	names := make([]string, 0)
	for name, _ := range tasks.Daemons {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		daemons := tasks.Daemons[name].Daemons
		cfg := program_map[name]

		fmt.Println("Program name:", name)
		fmt.Println("Cmd:         ", cfg.Cmd)
		for i, daemon := range daemons {
			status := ""
			if !daemon.NoRestart && daemon.Uptime != 0 && daemon.Uptime < int64(cfg.Starttime) {
				status = fmt.Sprintf("Starting %d/%d", daemon.Uptime, cfg.Starttime)
			} else if daemon.Stopping {
				status = "Stopping"
			} else if daemon.Running {
				status = "Running"
			} else {
				status = fmt.Sprintf("Exited (code=%d) %s", daemon.ExitCode, daemon.ErrMsg)
			}
			fmt.Printf("    Process %d:\n", i)
			fmt.Println("        Status:          ", status)
			if daemon.StartTime != 0 {
				fmt.Println("        Start time:      ", time.Unix(daemon.StartTime/tasks.SecondToMillisecond, 0))
			} else {
				fmt.Println("        Start time:       -")
			}
			fmt.Println("        Uptime (seconds):", daemon.Uptime)
			fmt.Println("        Start retries:   ", daemon.StartRetries, "/", cfg.Startretries)
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
	tasks.RegisterS("[ " + program_name + " ] restart")
	stop(program_name, cfg)
	tasks.RestartProgramAfterStopped(program_name, cfg)
}

func RemoveProgram(program_name string, cfg parse_yaml.Program) {
	stop(program_name, cfg)
	tasks.Remove(program_name)
}

func reload_config(program_map parse_yaml.ProgramMap, cfg_yaml string) {
	tasks.RegisterS("Reload config " + cfg_yaml)
	new_program_map, err := parse_yaml.ParseYaml(cfg_yaml)
	if err != nil {
		fmt.Println("Invalid config file: "+cfg_yaml+":", err)
		fmt.Println("reload_config failed, no config changes")
		log.Debug.Println("Invalid config file: "+cfg_yaml+":", err)
		log.Debug.Println("reload_config failed, no config changes")
		return
	}

	for key, cfg := range program_map {
		if _, key_exist := new_program_map[key]; !key_exist {
			handler := tasks.Daemons[key]
			handler.ToDelete = true
			handler.Stop()
		} else if !reflect.DeepEqual(cfg, new_program_map[key]) {
			handler := tasks.Daemons[key]
			handler.ReloadConfig()
			program_map[key] = new_program_map[key]
		}
	}

	for key, cfg := range new_program_map {
		if _, key_exist := program_map[key]; !key_exist {
			program_map[key] = new_program_map[key]
			tasks.Add(key, cfg)
			if cfg.Autostart {
				tasks.StartProgram(key, cfg)
			}
		}
	}
}

func exit(program_map parse_yaml.ProgramMap) {
	tasks.Stopping = true
	for key, cfg := range program_map {
		stop(key, cfg)
	}
}

func call_func(text string, program_map parse_yaml.ProgramMap, cfg_yaml string) {
	text_list := strings.Fields(text)

	if len(text_list) == 0 {
		return
	}

	cmd := text_list[0]
	args := text_list[1:]
	switch {
	case strings.HasPrefix("status", cmd):
		status(program_map)
	case strings.HasPrefix("start", cmd):
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
	case strings.HasPrefix("stop", cmd):
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
	case strings.HasPrefix("restart", cmd):
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
	case strings.HasPrefix("reload_config", cmd):
		tasks.StartMut.Lock()
		reload_config(program_map, cfg_yaml)
		tasks.StartMut.Unlock()
	case strings.HasPrefix("exit", cmd):
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

	var program_map parse_yaml.ProgramMap = nil

	program_map, err := parse_yaml.ParseYaml(cfg_yaml)
	if err != nil {
		l.Fatalln(err)
	}

	for name, program := range program_map {
		// fmt.Println("Creating program ", name)
		tasks.Add(name, program)
		if program.Autostart {
			// fmt.Println("Starting program ", name)
			tasks.StartProgram(name, program)
		}
	}
	// fmt.Println("Starting programs OK")

	// fmt.Println("Initializing signal handler")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	go func() {
		for {
			// Block until a signal is received.
			_ = <-c
			tasks.StartMut.Lock()
			reload_config(program_map, cfg_yaml)
			tasks.StartMut.Unlock()
		}
	}()
	// fmt.Println("Initializing signal handler OK")

	// fmt.Println("Launching master watch")
	go func() {
		for range time.Tick(1000 * time.Millisecond) {
			// fmt.Println("tasks.Stopping = ", tasks.Stopping)
			someAlive := false
			for name, handler := range tasks.Daemons {
				isAlive := master.WatchAlive(handler)
				handler.Started = isAlive
				if !isAlive {
					handler.Stopping = false
				}
				if !isAlive && handler.ToDelete {
					tasks.Remove(name)
					delete(program_map, name)
				} else if !isAlive && handler.ToReload && !tasks.Stopping {
					handler.Init(name, program_map[name])
					if handler.Cfg.Autostart {
						handler.Start()
					}
				}
				someAlive = someAlive || isAlive
			}
			if tasks.Stopping && !someAlive {
				os.Exit(0)
			}
		}
	}()
	// fmt.Println("Launching master watch OK")

	term.Init()

	for {
		text := term.ReadLine()
		call_func(text, program_map, cfg_yaml)
	}
}
