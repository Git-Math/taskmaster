package tasks

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"taskmaster/parse_yaml"
	"time"
)

var SecondToMillisecond = int64(1000)

func CurrentTimeMillisecond() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

/* {{{ Register */

var registerFile io.Writer = nil

func Register(daemon *Daemon, msg string) {
	if registerFile == nil {
		var err error

		registerFile, err = os.Create("register.log")
		if err != nil {
			log.Fatal(err)
		}
	}
	fmt.Fprintln(registerFile, CurrentTimeMillisecond(), "[", daemon.Name, "]", msg)
	fmt.Println(CurrentTimeMillisecond(), "[", daemon.Name, "]", msg)
}

/* }}} */

type Daemon struct {
	Name         string     /* name of the program from the yaml */
	Command      *exec.Cmd  /* Cmd */
	Dead         bool       /* Indicate that the daemon is dead */
	StartTime    int64      /* Start Time of the program */
	StartRetries int        /* Count of time the program was restarted because it stopped before Starttime */
	Running      bool       /* Indicate that the program has been running long enough to say it's running */
	ExitCode     int        /* Exit Code of the program or -1 */
	Err          chan error /* Channel to the goroutine waiting for the program to return */
}

var Mutex sync.Mutex
var Daemons = make(map[string]([]*Daemon))

func DaemonRetrieve(name string) []*Daemon {
	for _, daemons := range Daemons {
		if daemons[0].Name == name {
			return daemons
		}
	}
	return nil
}

func (dae *Daemon) Start(cfg parse_yaml.Program) {
	dae.Reset()

	command_parts := strings.Fields(cfg.Cmd)
	if len(command_parts) > 1 {
		dae.Command = exec.Command(command_parts[0], command_parts[1:len(command_parts)]...)
	} else {
		dae.Command = exec.Command(command_parts[0])
	}

	dae.Command.Env = cfg.Env
	dae.Command.Dir = cfg.Workingdir

	if cfg.Stdout != "" {
		outfile, err := os.Create(cfg.Stdout)
		if err != nil {
			log.Fatal(err)
		}
		dae.Command.Stdout = outfile
	} else {
		dae.Command.Stdout = nil
	}

	if cfg.Stderr != "" {
		errfile, err := os.Create(cfg.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		dae.Command.Stderr = errfile
	} else {
		dae.Command.Stderr = nil
	}

	dae.StartRetries++
	dae.StartTime = CurrentTimeMillisecond()
	dae.Err = make(chan error)
	dae.Err <- dae.Command.Run()
}

func (dae *Daemon) Stop() {
	if !dae.Command.ProcessState.Exited() {
		log.Fatal("dev: Called daemon.Stop before process `" + dae.Name + "' exited")
	}
	dae.Reset()
	dae.ExitCode = dae.Command.ProcessState.ExitCode()
}

func (dae *Daemon) Reset() {
	dae.Running = false
	dae.ExitCode = -1
}

func (dae *Daemon) IsRunning() bool {
	Mutex.Lock()
	running := dae.Running
	Mutex.Unlock()
	return running
}

func StartProgram(name string, cfg parse_yaml.Program) {
	for _, daemon := range Daemons[name] {
		go daemon.Start(cfg)
	}
}

func StopProgram(cfg parse_yaml.Program, daemon *Daemon) {
	Mutex.Lock()
	daemon.Command.Process.Signal(parse_yaml.SignalMap[cfg.Stopsignal])
	Mutex.Unlock()

	time.Sleep(time.Duration(cfg.Stoptime) * time.Second)

	Mutex.Lock()
	if daemon.Running {
		daemon.Command.Process.Kill()
	}
	Mutex.Unlock()

	daemon.Stop()
}

func Add(name string, cfg parse_yaml.Program) {
	Daemons[name] = []*Daemon{}
	for i := 0; i < cfg.Numprocs; i++ {
		var daemon Daemon

		daemon.Name = name
		Daemons[name] = append(Daemons[name], &daemon)
	}
}
