package master

import (
	"strconv"
	"taskmaster/log"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

var Stopping bool = false

func watchDaemon(dae *tasks.Daemon, cfg parse_yaml.Program) {
	dae.Lock()

	if dae.NoRestart || dae.StartTime == 0 {
		dae.Unlock()
		return
	}

	dae.Uptime = (tasks.CurrentTimeMillisecond() - dae.StartTime) / 1000
	exited := false
	var err error = nil

	select {
	case err = <-dae.Err:
		exited = true
		dae.Running = false
		if err != nil {
			dae.ErrMsg = err.Error()
		} else {
			dae.ExitCode = 0
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
					return
				}
				exited = true
				dae.Running = false
				dae.ErrMsg = "forced stop"
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

		if dae.StartRetries < cfg.Startretries {
			go dae.Start(cfg)
		}

		return
	}

	if exited {
		if dae.ErrMsg != "" {
			tasks.Register(dae, "Exited ("+dae.ErrMsg+")")
		} else {
			tasks.Register(dae, "Exited ("+"Success"+")")
		}

		restart := false
		switch cfg.Autorestart {
		case "unexpected":
			success := false
			for exitSuccess := range cfg.Exitcodes {
				if exitSuccess == dae.ExitCode {
					success = true
					break
				}
			}
			if !success {
				restart = true
			}
		case "always":
			restart = true
		case "never":
		}

		if !restart {
			dae.NoRestart = true
		}

		// unlock before calling Start
		dae.StoptimeCounter = 0
		dae.Stopping = false
		dae.StartTime = 0

		if !dae.Stopping && restart {
			dae.StartRetries = 0
			tasks.Register(dae, "restarting ..")
			dae.Unlock()
			go dae.Start(cfg)
			return
		}

		dae.Unlock()

		return
	}

	if !dae.Running && dae.Uptime >= int64(cfg.Starttime) {
		/* dae just passed StartTime */

		tasks.Register(dae, "Started")
		dae.Running = true
	}

	dae.Unlock()
}

func Watch(programs_cfg parse_yaml.ProgramMap) {
	for range time.Tick(1000 * time.Millisecond) {
		if Stopping {
			break
		}
		for _, daemons := range tasks.Daemons {
			for _, daemon := range daemons {
				cfg := programs_cfg[daemon.Name]

				if !daemon.NoRestart {
					log.Debug.Println("Watching for daemon", daemon.Name)
					watchDaemon(daemon, cfg)
					log.Debug.Println("Watching for daemon", daemon.Name, "returned")
				}
			}
		}
	}
}
