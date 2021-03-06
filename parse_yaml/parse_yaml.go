package parse_yaml

import (
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strconv"
	"syscall"
)

var SignalMap = map[string]syscall.Signal{
	"HUP":  syscall.SIGHUP,
	"INT":  syscall.SIGINT,
	"QUIT": syscall.SIGQUIT,
	"ILL":  syscall.SIGILL,
	"ABRT": syscall.SIGABRT,
	"FPE":  syscall.SIGFPE,
	"KILL": syscall.SIGKILL,
	"SEGV": syscall.SIGSEGV,
	"PIPE": syscall.SIGPIPE,
	"ALRM": syscall.SIGALRM,
	"TERM": syscall.SIGTERM,
	"USR1": syscall.SIGUSR1,
	"USR2": syscall.SIGUSR2,
	"CHLD": syscall.SIGCHLD,
	"CONT": syscall.SIGCONT,
	"STOP": syscall.SIGSTOP,
	"TSTP": syscall.SIGTSTP,
	"TTIN": syscall.SIGTTIN,
	"TTOU": syscall.SIGTTOU,
}

type Program struct {
	Cmd          string   `yaml:"cmd"`
	Numprocs     int      `yaml:"numprocs"`
	Umask        string   `yaml:"umask"`
	Workingdir   string   `yaml:"workingdir"`
	Autostart    bool     `yaml:"autostart"`
	Autorestart  string   `yaml:"autorestart"`
	Exitcodes    []int    `yaml:"exitcodes"`
	Startretries int      `yaml:"startretries"`
	Starttime    int      `yaml:"starttime"`
	Stopsignal   string   `yaml:"stopsignal"`
	Stoptime     int      `yaml:"stoptime"`
	Stdout       string   `yaml:"stdout"`
	Stderr       string   `yaml:"stderr"`
	Env          []string `yaml:"env"`
}

type ProgramMap = map[string]Program

type ProgramsStruct struct {
	Programs ProgramMap `yaml:"programs"`
}

func CheckUmask(umask string) bool {
	for i := 0; i < len(umask); i++ {
		if umask[i] < '0' || umask[i] > '7' {
			return false
		}
	}
	return true
}

func CheckYaml(program_map ProgramMap) error {
	for key, program := range program_map {
		if program.Cmd == "" {
			return errors.New("Program " + key + ": invalid cmd value: can't be empty")
		}
		if program.Numprocs < 1 {
			return errors.New("Program " + key + ": invalid numprocs value: [" + strconv.Itoa(program.Numprocs) + "], should be greater than 0")
		}
		if len(program.Umask) != 3 {
			return errors.New("Program " + key + ": invalid umask size: [" + strconv.Itoa(len(program.Umask)) + "], should be 3")
		}
		if !CheckUmask(program.Umask) {
			return errors.New("Program " + key + ": invalid umask value: [" + program.Umask + "], must contain only digits between 0 and 7")
		}
		if program.Autorestart != "always" && program.Autorestart != "never" && program.Autorestart != "unexpected" {
			return errors.New("Program " + key + ": invalid autorestart value: [" + program.Autorestart + "], should be [always], [never] or [unexpected]")
		}
		if len(program.Exitcodes) == 0 {
			return errors.New("Program " + key + ": invalid exitcodes value: can't be empty")
		}
		if program.Startretries < 0 {
			return errors.New("Program " + key + ": invalid startretries value: [" + strconv.Itoa(program.Startretries) + "], should be greater than or equal to 0")
		}
		if program.Starttime < 0 {
			return errors.New("Program " + key + ": invalid starttime value: [" + strconv.Itoa(program.Starttime) + "], should be greater than or equal to 0")
		}
		if program.Stoptime < 1 {
			return errors.New("Program " + key + ": invalid stoptime value: [" + strconv.Itoa(program.Stoptime) + "], should be greater than 0")
		}
		if _, key_exist := SignalMap[program.Stopsignal]; !key_exist {
			return errors.New("Program " + key + ": invalid stopsignal value: [" + program.Stopsignal + "]")
		}
	}
	return nil
}

func ParseYaml(yamlfile string) (ProgramMap, error) {
	var programs_struct ProgramsStruct

	yaml_data, err := ioutil.ReadFile(yamlfile)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yaml_data, &programs_struct)
	if err != nil {
		return nil, err
	}

	err = CheckYaml(programs_struct.Programs)
	if err != nil {
		return nil, err
	}

	return programs_struct.Programs, nil
}
