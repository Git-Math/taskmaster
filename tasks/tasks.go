package tasks

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"taskmaster/debug"
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
	NoRestart    bool       /* Indicate that the daemon is dead */
	StartTime    int64      /* Start Time of the program */
	StartRetries int        /* Count of time the program was restarted because it stopped before Starttime */
	Running      bool       /* Indicate that the program has been running long enough to say it's running */
	ExitCode     int        /* Exit Code of the program or -1 */
	Err          chan error /* Channel to the goroutine waiting for the program to return */
	mut          sync.Mutex
}

var Daemons = make(map[string]([]*Daemon))

func DaemonRetrieve(name string) []*Daemon {
	for _, daemons := range Daemons {
		if daemons[0].Name == name {
			return daemons
		}
	}
	return nil
}

func (dae *Daemon) Lock() {
	debug.DebugLog.Println("locking", dae.Name)
	dae.mut.Lock()
	debug.DebugLog.Println("locked", dae.Name)
}

func (dae *Daemon) Unlock() {
	debug.DebugLog.Println("unlocking", dae.Name)
	dae.mut.Unlock()
	debug.DebugLog.Println("unlocked", dae.Name)
}

func (dae *Daemon) Start(cfg parse_yaml.Program) {
	debug.DebugLog.Println("Start daemon", cfg.Cmd)
	dae.Lock()

	dae.reset()

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

	dae.Unlock()

	dae.Err = make(chan error)
	dae.Err <- dae.Command.Run()
}

func (dae *Daemon) stop() {
	if !dae.Command.ProcessState.Exited() {
		log.Fatal("dev: Called daemon.Stop before process `" + dae.Name + "' exited")
	}
	dae.reset()
	dae.ExitCode = dae.Command.ProcessState.ExitCode()
}

func (dae *Daemon) reset() {
	dae.Running = false
	dae.ExitCode = -1
}

func StartProgram(name string, cfg parse_yaml.Program) {
	for _, daemon := range Daemons[name] {
		go daemon.Start(cfg)
	}
}

func StopProgram(program_name string, cfg parse_yaml.Program) {
	var wg sync.WaitGroup

	daemons := DaemonRetrieve(program_name)
	for _, dae := range daemons {
		dae.Lock()
		running := dae.Running
		dae.Unlock()
		fmt.Println("Stopping", program_name, "running=", running)
		if running {
			wg.Add(1)
			go func() {
				defer wg.Done()

				fmt.Println(dae.Name, "stopping ...")

				dae.Lock()
				dae.Command.Process.Signal(parse_yaml.SignalMap[cfg.Stopsignal])
				dae.Unlock()

				time.Sleep(time.Duration(cfg.Stoptime) * time.Second)

				dae.Lock()
				if dae.Running {
					dae.Command.Process.Kill()
				}
				dae.stop()
				dae.Unlock()
			}()
		}
	}
	wg.Wait()
}

func Add(name string, cfg parse_yaml.Program) {
	Daemons[name] = []*Daemon{}
	for i := 0; i < cfg.Numprocs; i++ {
		var dae Daemon

		dae.Name = name
		Daemons[name] = append(Daemons[name], &dae)
	}
}
