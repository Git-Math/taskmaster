package tasks

import (
	"fmt"
	l "log"
	"os"
	"os/exec"
	"strconv"
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

var registerFile *os.File = nil

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

func RegisterS(msg string) {
	if registerFile == nil {
		var err error

		registerFile, err = os.Create("register.log")
		if err != nil {
			l.Fatal(err)
		}
	}
	fmt.Fprintln(registerFile, CurrentTimeMillisecond(), msg)
	log.Debug.Println(CurrentTimeMillisecond(), msg)
}

/* }}} */

type Daemon struct {
	Name                   string             /* name of the program from the yaml */
	Command                *exec.Cmd          /* Cmd */
	NoRestart              bool               /* Indicate that the daemon is dead */
	StartTime              int64              /* Start Time of the program */
	Uptime                 int64              /* Uptime */
	StartRetries           int                /* Count of time the program was restarted because it stopped before Starttime */
	Running                bool               /* Indicate that the program has been running long enough to say it's running */
	Stopping               bool               /* program is stopping */
	RestartAfterStopped    bool               /* indicate the program should restart once it's stopped */
	RestartAfterStoppedCfg parse_yaml.Program /* indicate the program should restart once it's stopped */
	StoptimeCounter        int                /* count before stoptime */
	ExitCode               int                /* Exit Code of the program or -1 */
	Err                    chan error         /* Channel to the goroutine waiting for the program to return */
	ErrMsg                 string
	mut                    sync.Mutex
}

type DaemonHandler struct {
	Name     string
	Daemons  []*Daemon
	Cfg      parse_yaml.Program
	Started  bool
	Stopping bool
	ToDelete bool
	ToReload bool
}

func (h *DaemonHandler) Init(name string, cfg parse_yaml.Program) {
	h.Name = name
	h.Daemons = []*Daemon{}
	for i := 0; i < cfg.Numprocs; i++ {
		var dae Daemon

		dae.Name = name
		dae.reset()
		dae.NoRestart = false
		dae.StartTime = 0
		dae.StartRetries = 0
		h.Daemons = append(h.Daemons, &dae)
	}
	h.Cfg = cfg
	h.Started = false
	h.Stopping = false
	h.ToDelete = false
	h.ToReload = false
}

func (h *DaemonHandler) Start() {
	// fmt.Println(h.Name, "handler START")
	for _, dae := range h.Daemons {
		dae.reset()
		dae.NoRestart = false
		dae.StartTime = 0
		dae.StartRetries = 0
		go dae.Start(h.Cfg)
	}
	h.Stopping = false
	h.ToDelete = false
	h.ToReload = false

	h.Started = true
}

func (h *DaemonHandler) Stop() {
	h.Stopping = true
	for _, dae := range h.Daemons {
		dae.Stop(h.Cfg)
	}
}

func (h *DaemonHandler) ReloadConfig() {
	h.ToReload = true
	h.Stop()
}

var Daemons = make(map[string]*DaemonHandler)

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
	dae.Stopping = false
	dae.StoptimeCounter = 0
	dae.ExitCode = -1
	dae.Err = nil
}

var StartMut sync.Mutex
var Stopping bool = false

func (dae *Daemon) Start(cfg parse_yaml.Program) {
	StartMut.Lock()
	log.Debug.Println("Start daemon", cfg.Cmd)

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
			StartMut.Unlock()
			l.Fatal(err)
		}
		dae.Command.Stdout = outfile
	} else {
		dae.Command.Stdout = nil
	}

	if cfg.Stderr != "" {
		errfile, err := os.Create(cfg.Stderr)
		if err != nil {
			StartMut.Unlock()
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

	umask, err := strconv.ParseUint(cfg.Umask, 8, 0)
	if err != nil {
		log.Debug.Printf("%s: invalid umask `%s`: %v. umask ignored\n", dae.Name, cfg.Umask, err)
		umask = 2
	}
	old_mask := syscall.Umask(int(umask))

	if err := dae.Command.Start(); err != nil {
		log.Debug.Printf("%s: failed to execute command `%s`: %v\n", dae.Name, cfg.Cmd, err)
		dae.reset()
		dae.ExitCode = -1
		dae.ErrMsg = fmt.Sprintf("failed to execute command `%s`: %v\n", cfg.Cmd, err)
		dae.NoRestart = true
		_ = syscall.Umask(old_mask)
		StartMut.Unlock()
		return
	}

	_ = syscall.Umask(old_mask)

	StartMut.Unlock()

	dae.Err = make(chan error)
	dae.Err <- dae.Command.Wait()
}

func (dae *Daemon) Stop(cfg parse_yaml.Program) {
	dae.Lock()
	running := !dae.Stopping && (dae.Running || dae.StartTime > 0)
	log.Debug.Println("Stopping", dae.Name, "running=", running)
	if running {
		log.Debug.Println(dae.Name, "stopping ...")
		dae.Stopping = true
		dae.StoptimeCounter = 0
		err := dae.Command.Process.Signal(parse_yaml.SignalMap[cfg.Stopsignal])
		if err != nil {
			log.Debug.Println(dae.Name, ": failed to stop program cleanly:", err)
		}
	}
	dae.Unlock()
}

func StartProgram(name string, cfg parse_yaml.Program) {
	handler := Daemons[name]
	fmt.Println("Program", name, "started?", handler.Started, "stopping?", handler.Stopping)
	if handler.Started || handler.Stopping {
		return
	}
	handler.Start()
	fmt.Println("Program", name, "started?", handler.Started, "stopping?", handler.Stopping)
}

func StopProgram(program_name string, cfg parse_yaml.Program) {
	handler := Daemons[program_name]
	if !handler.Started || handler.Stopping {
		return
	}
	handler.Stop()
}

func RestartProgramAfterStopped(program_name string, cfg parse_yaml.Program) {
	handler := Daemons[program_name]
	for _, dae := range handler.Daemons {
		dae.Lock()
		if dae.Stopping {
			dae.RestartAfterStopped = true
		}
		dae.Unlock()
	}
}

func Add(name string, cfg parse_yaml.Program) {
	Daemons[name] = &DaemonHandler{}
	Daemons[name].Init(name, cfg)
}

func Remove(name string) {
	delete(Daemons, name)
}
