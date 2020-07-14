package tasks

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"taskmaster/parse_yaml"
	"time"
)

var SecondToMillisecond = int64(1000)

func CurrentTimeMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

type Daemon struct {
	Name      string     /* name of the program from the yaml */
	Command   *exec.Cmd  /* Cmd */
	StartTime int64      /* Start Time of the program */
	Running   bool       /* Indicate that the program has been running long enough to say it's running */
	ExitCode  int        /* Exit Code of the program or -1 */
	Err       chan error /* Channel to the goroutine waiting for the program to return */
}

func (dae *Daemon) Start() error {
	dae.reset()
	err := dae.Command.Start()
	if err != nil {
		return err
	}
	dae.Err = make(chan error)
	dae.Err <- dae.Command.Run()
	return nil
}

func (dae *Daemon) Stop() {
	if !dae.Command.ProcessState.Exited() {
		log.Fatal("Called Stop() but process `" + dae.Name + "'did not exit")
	}
	dae.reset()
	dae.ExitCode = dae.Command.ProcessState.ExitCode()
}

func (dae *Daemon) reset() {
	dae.StartTime = 0
	dae.Running = false
	dae.ExitCode = -1
	dae.Err = nil
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
			if err == nil {
				daemon.StartTime = CurrentTimeMillisecond()
				time.Sleep(time.Duration(startTime) * time.Second)
				if !daemon.Command.ProcessState.Exited() {
					Register(daemon, "Started")
					daemon.Running = true
					return
				}
				daemon.reset()
			}
			/* FIXME: what if the program exited successfully already ? */
			if retrieCount == 0 {
				fmt.Println("Start time=", startTime)
				msg := "Failed to start after " + strconv.Itoa(startTime) + " times"
				if err != nil {
					msg += ": " + err.Error()
				}
				Register(daemon, msg)
				break
			} else {
				retrieCount--
			}
		}
	}

	go startDaemon(cfg.Startretries, cfg.Starttime)
}

func StopProgram(cfg parse_yaml.Program, daemon *Daemon) {
	go func() {
		daemon.Command.Process.Signal(parse_yaml.SignalMap[cfg.Stopsignal])
		time.Sleep(time.Duration(cfg.Stoptime) * time.Second)
		if daemon.Running {
			daemon.Command.Process.Kill()
		}
		daemon.Stop()
	}()
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
