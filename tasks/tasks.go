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

type Daemon struct {
	Name           string
	Command        *exec.Cmd
	StartTimestamp time.Time
	Running        bool
	StartTime      int64
	StartRepeat    int
	Err            chan error
}

var Daemons []*Daemon

var second_to_milisecond = int64(1000)

func CurrentTimeMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func Register(daemon *Daemon, date time.Time, msg string) {
	registerFile, err := os.OpenFile("register.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer registerFile.Close()

	fmt.Fprintln(registerFile, date, "[", daemon.Name, "]", msg)
}

func StartProgram(daemon *Daemon) {
	go func() {
		date := time.Now()

		err := daemon.Command.Start()
		if err != nil {
			Register(daemon, date, "Failed to start")
			return
		}
		Register(daemon, date, "Started")
		daemon.Running = true
		daemon.Err = make(chan error)
		daemon.Err <- daemon.Command.Run()
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
				StartProgram(&daemon)
			}
		}
	}
}
