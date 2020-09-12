package master

import (
	"strconv"
	"taskmaster/log"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func watchDaemon(dae *tasks.Daemon, cfg parse_yaml.Program) {
	dae.Lock()

	if dae.NoRestart {
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
			tasks.Register(dae, "Exited ("+dae.ErrMsg+")")
		} else {
			tasks.Register(dae, "Exited ("+"Success"+")")
			dae.ExitCode = 0
		}
	default:
	}

	log.Debug.Println(dae.Name, "uptime =", dae.Uptime, "start-time =", cfg.Starttime)

	if exited && dae.Uptime < int64(cfg.Starttime) {
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
		dae.Unlock()

		if restart {
			go dae.Start(cfg)
		}

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
