package tasks

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"taskmaster/parse_yaml"
	"time"
)

var second_to_milisecond = int64(1000)

func CurrentTimeMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

type Daemon struct {
	Name      string     /* name of the program from the yaml */
	Command   *exec.Cmd  /* Cmd */
	StartTime int64      /* Start Time of the program */
	Running   bool       /* Indicate that the program has been running long enough to say it's running */
	Err       chan error /* Channel to the goroutine waiting for the program to return */
}

func (dae *Daemon) Start() error {
	err := dae.Command.Start()
	if err != nil {
		return err
	}
	dae.Running = false
	dae.Err = make(chan error)
	dae.Err <- dae.Command.Run()
	return nil
}

func (dae *Daemon) Run() {
	dae.Running = true
}

func (dae *Daemon) Stop() {
	dae.Running = false
	dae.StartTime = 0
}

var Daemons []*Daemon

func Register(daemon *Daemon, msg string) {
	date := CurrentTimeMillisecond()
	registerFile, err := os.OpenFile("register.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer registerFile.Close()

	fmt.Fprintln(registerFile, date, "[", daemon.Name, "]", msg)
}

func StartProgram(cfg parse_yaml.Program, daemon *Daemon) {

	startDaemon := func(retrieCount int, startTime int) {
		for {
			err := daemon.Start()
			if err != nil {
				daemon.StartTime = CurrentTimeMillisecond()
				time.Sleep(time.Duration(startTime) * time.Second)
				if !daemon.Command.ProcessState.Exited() {
					Register(daemon, "Started")
					daemon.Run()
					return
				}
				daemon.Stop()
			}
			if retrieCount == 0 {
				Register(daemon, "Failed to start after"+string(startTime)+"times")
				break
			} else {
				retrieCount--
			}
		}
	}

	go startDaemon(cfg.Startretries, cfg.Starttime)
}

func Execute(program_map parse_yaml.ProgramMap) {
	fmt.Println("I execute the processes and store Cmds in Daemons")
	for key, program := range program_map {
		for i := 0; i < program.Numprocs; i++ {
			var daemon Daemon

			daemon.Name = key

			command_parts := strings.Fields(program.Cmd)
			if len(command_parts) > 1 {
				daemon.Command = exec.Command(command_parts[0], command_parts[1:len(command_parts)]...)
			} else {
				daemon.Command = exec.Command(command_parts[0])
			}

			daemon.Command.Env = program.Env

			daemon.Command.Dir = program.Workingdir

			if program.Stdout != "" {
				outfile, err := os.Create(program.Stdout)
				if err != nil {
					log.Fatal(err)
				}
				defer outfile.Close()
				daemon.Command.Stdout = outfile
			} else {
				daemon.Command.Stdout = nil
			}

			if program.Stderr != "" {
				errfile, err := os.Create(program.Stderr)
				if err != nil {
					log.Fatal(err)
				}
				defer errfile.Close()
				daemon.Command.Stderr = errfile
			} else {
				daemon.Command.Stderr = nil
			}

			Daemons = append(Daemons, &daemon)

			if program.Autostart {
				StartProgram(program, &daemon)
			}
		}
	}
}
