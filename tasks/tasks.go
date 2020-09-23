package tasks

import (
	"fmt"
	"io"
	l "log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"taskmaster/log"
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
			l.Fatal(err)
		}
	}
	fmt.Fprintln(registerFile, CurrentTimeMillisecond(), "[", daemon.Name, "]", msg)
	log.Debug.Println(CurrentTimeMillisecond(), "[", daemon.Name, "]", msg)
}

/* }}} */

type Daemon struct {
	Name         string     /* name of the program from the yaml */
	Command      *exec.Cmd  /* Cmd */
	NoRestart    bool       /* Indicate that the daemon is dead */
	StartTime    int64      /* Start Time of the program */
	Uptime       int64      /* Uptime */
	StartRetries int        /* Count of time the program was restarted because it stopped before Starttime */
	Running      bool       /* Indicate that the program has been running long enough to say it's running */
	ExitCode     int        /* Exit Code of the program or -1 */
	Err          chan error /* Channel to the goroutine waiting for the program to return */
	ErrMsg       string
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
	log.Debug.Println("locking", dae.Name)
	dae.mut.Lock()
	log.Debug.Println("locked", dae.Name)
}

func (dae *Daemon) Unlock() {
	log.Debug.Println("unlocking", dae.Name)
	dae.mut.Unlock()
	log.Debug.Println("unlocked", dae.Name)
}

func (dae *Daemon) reset() {
	dae.StartTime = 0
	dae.Uptime = 0
	dae.Running = false
	dae.ExitCode = -1
	dae.Err = nil
}

func (dae *Daemon) Init() {
	dae.Lock()
	dae.reset()
	dae.NoRestart = false
	dae.StartTime = 0
	dae.StartRetries = 0
	dae.Unlock()
}

func (dae *Daemon) Start(cfg parse_yaml.Program) {
	log.Debug.Println("Start daemon", cfg.Cmd)
	dae.Lock()

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
			l.Fatal(err)
		}
		dae.Command.Stdout = outfile
	} else {
		dae.Command.Stdout = nil
	}

	if cfg.Stderr != "" {
		errfile, err := os.Create(cfg.Stderr)
		if err != nil {
			l.Fatal(err)
		}
		dae.Command.Stderr = errfile
	} else {
		dae.Command.Stderr = nil
	}

	dae.Command.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: parse_yaml.SignalMap[cfg.Stopsignal],
	}

	dae.StartRetries++
	dae.StartTime = CurrentTimeMillisecond()

	dae.Unlock()

	dae.Err = make(chan error)
	dae.Err <- dae.Command.Run()
}

func (dae *Daemon) stop() {
	if !dae.Command.ProcessState.Exited() {
		l.Fatal("dev: Called daemon.Stop before process `" + dae.Name + "' exited")
	}
	dae.ExitCode = dae.Command.ProcessState.ExitCode()
}

func StartProgram(name string, cfg parse_yaml.Program) {
	for _, daemon := range Daemons[name] {
		daemon.Init()
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
		log.Debug.Println("Stopping", program_name, "running=", running)
		if running {

			log.Debug.Println(dae.Name, "stopping ...")
			dae.Lock()
			err := dae.Command.Process.Signal(parse_yaml.SignalMap[cfg.Stopsignal])
			dae.Unlock()
			if err != nil {
				log.Debug.Println(dae.Name, ": failed to stop program cleanly:", err)
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				exited := false
				for i := 0; i < cfg.Stoptime; i++ {
					time.Sleep(1 * time.Second)
					dae.Lock()
					errmsg := dae.Command.ProcessState.String()
					dae.Unlock()
					if errmsg != "<nil>" {
						exited = true
						dae.ErrMsg = errmsg
						break
					}
				}

				if !exited {
					log.Debug.Println(dae.Name, ": failed to stop program cleanly, now forcing ..")
					dae.Lock()
					err = dae.Command.Process.Kill()
					dae.Unlock()
					if err != nil {
						log.Debug.Println(dae.Name, ": failed to stop program:", err)
						return
					}
				}

				dae.Lock()
				dae.reset()
				dae.ExitCode = 0
				dae.ErrMsg = dae.Command.ProcessState.String()
				dae.NoRestart = true
				dae.Unlock()

				log.Debug.Println(dae.Name, "stopped")
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

func Remove(name string) {
	delete(Daemons, name)
}
