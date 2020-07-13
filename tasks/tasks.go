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
	Restart        int
	err            chan error
}

var Daemons []Daemon

func StartProgram(daemon Daemon) {
	daemon.err <- daemon.Command.Run()
	select {
	default:
		fmt.Println("process is running")
	case e := <-daemon.err:
		if e != nil {
			fmt.Println("process exited: ", e)
		} else {
			fmt.Println("process exited successfully")
		}
	}
}

func Execute(program_map parse_yaml.ProgramMap) {
	fmt.Println("I execute the processes and store Cmds in Daemons")
	for key, program := range program_map {
		for i := 0; i < program.Numprocs; i++ {
			var daemon Daemon

			daemon.Name = key
			daemon.Restart = 0

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

			if program.Autostart {
				StartProgram(daemon)
			}

			Daemons = append(Daemons, daemon)
		}
	}
}
