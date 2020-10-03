package master

import (
	"strconv"
	"syscall"
	"taskmaster/log"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
)

var Stopping bool = false

// return true if the daemon is running, false otherwise
func watchDaemon(dae *tasks.Daemon, cfg parse_yaml.Program) bool {
	dae.Lock()

	if dae.NoRestart || dae.StartTime == 0 {
		dae.Unlock()
		return false
	}

	log.Debug.Println("Watching for daemon", dae.Name, dae.Instance)

	dae.Uptime = (tasks.CurrentTimeMillisecond() - dae.StartTime) / 1000
	exited := false
	var err error = nil

	select {
	case err = <-dae.Err:
		exited = true
		dae.Running = false
		if err != nil {
			dae.ErrMsg = err.Error()
			dae.ExitCode = int(dae.Command.ProcessState.Sys().(syscall.WaitStatus))
		} else {
			dae.ExitCode = dae.Command.ProcessState.ExitCode()
		}
	default:
	}

	log.Debug.Println(dae.Name, "uptime =", dae.Uptime, "start-time =", cfg.Starttime)

	if !exited && dae.Stopping {
		errmsg := dae.Command.ProcessState.String()
		if errmsg != "<nil>" {
			exited = true
			dae.Running = false
			dae.ErrMsg = errmsg
			dae.ExitCode = int(dae.Command.ProcessState.Sys().(syscall.WaitStatus))
		} else {
			dae.StoptimeCounter++
			if dae.StoptimeCounter == cfg.Stoptime {
				err = dae.Command.Process.Kill()
				if err != nil {
					tasks.Register(dae, "failed to stop: "+err.Error())
					log.Debug.Println(dae.Name, ": failed to stop program:", err)
					dae.StoptimeCounter = 0
					dae.Stopping = false
					dae.Unlock()
					return true
				}
				exited = true
				dae.Running = false
				dae.ErrMsg = "forced stop"
				dae.ExitCode = int(syscall.SIGKILL)
			}
		}
	}

	if !dae.Stopping && exited && dae.Uptime < int64(cfg.Starttime) {
		/* dae exited before StartTime, check if it needs restarting */
		log.Debug.Println(dae.Name, "exited=", exited, "startRetries=", dae.StartRetries, "max=", cfg.Startretries)

		if dae.StartRetries == cfg.Startretries {
			msg := "Failed to start after " + strconv.Itoa(cfg.Startretries) + " times"
			if err != nil {
				msg += ": " + err.Error()
			}
			tasks.Register(dae, msg)
			dae.NoRestart = true
		}

		// unlock before calling Start
		dae.Unlock()

		if dae.StartRetries < cfg.Startretries && !tasks.Stopping {
			go dae.Start(cfg)
		}

		return false
	}

	if exited {
		if dae.ErrMsg != "" {
			tasks.Register(dae, "Exited ("+dae.ErrMsg+")")
		} else {
			tasks.Register(dae, "Exited ("+"Success"+")")
		}

		unexpected := true
		for _, exitSuccess := range cfg.Exitcodes {
			if exitSuccess == dae.ExitCode {
				unexpected = false
				break
			}
		}
		if unexpected {
			tasks.Register(dae, "Unexpected stop, exit code: "+strconv.Itoa(dae.ExitCode))
		}

		restart := false
		switch cfg.Autorestart {
		case "unexpected":
			success := false
			if !unexpected {
				success = true
			}
			if !success {
				restart = true
			}
		case "always":
			restart = true
		case "never":
		}

		restart = restart || dae.RestartAfterStopped
		dae.RestartAfterStopped = false

		if !restart {
			dae.NoRestart = true
		}

		dae.StoptimeCounter = 0
		dae.Stopping = false
		dae.StartTime = 0

		if !tasks.Stopping && restart {
			dae.StartRetries = 0
			dae.Unlock()
			tasks.Register(dae, "restarting ..")
			go dae.Start(cfg)
			return false
		}

		dae.Unlock()

		return false
	}

	if !dae.Running && dae.Uptime >= int64(cfg.Starttime) {
		/* dae just passed StartTime */

		tasks.Register(dae, "Started")
		dae.Running = true
	}

	dae.Unlock()

	return true
}

func WatchAlive(handler *tasks.DaemonHandler) bool {
	// fmt.Println("Watching", handler.Name)
	alive := false
	for _, daemon := range handler.Daemons {
		alive = watchDaemon(daemon, handler.Cfg) || alive
	}
	return alive
}
