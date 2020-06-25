package term

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"unicode"
	"unicode/utf8"
)

type Cmd struct {
	text  []byte
	index int
}

type Internal struct {
	history       []Cmd
	history_index int
}

var internal = Internal{}

func Init() {
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
}

/* key mapping */

func enter(cmd *Cmd) {
	if len(cmd.text) == 0 {
		return
	}

	to_append := false
	if len(internal.history) == 0 {
		to_append = true
	} else {
		if internal.history_index >= len(internal.history) {
			to_append = string(cmd.text) != string(internal.history[len(internal.history)-1].text)
		} else {
			to_append = string(cmd.text) != string(internal.history[internal.history_index].text)
		}
	}

	if to_append {
		internal.history = append(internal.history, Cmd{cmd.text, len(cmd.text)})
	}
	internal.history_index = len(internal.history)
}

func backspace(cmd *Cmd) {
	if cmd.index == 0 {
		return
	}
	copy(cmd.text[cmd.index-1:], cmd.text[cmd.index:])
	cmd.text = cmd.text[:len(cmd.text)-1]
	cmd.index--
}

func arrow_up(cmd *Cmd) {
	if internal.history_index == 0 {
		return
	}
	internal.history_index--
	*cmd = internal.history[internal.history_index]
}

func arrow_down(cmd *Cmd) {
	if internal.history_index >= len(internal.history)-1 {
		internal.history_index = len(internal.history)
		*cmd = Cmd{}
		return
	}
	internal.history_index++
	*cmd = internal.history[internal.history_index]
}

func arrow_left(cmd *Cmd) {
	if cmd.index == 0 {
		return
	}
	cmd.index--
}

func arrow_right(cmd *Cmd) {
	if cmd.index == len(cmd.text) {
		return
	}
	cmd.index++
}

var keyMap = map[uint32]interface{}{
	167772160:  enter,
	2130706432: backspace,
	458965248:  arrow_up,
	458965504:  arrow_down,
	458966016:  arrow_left,
	458965760:  arrow_right,
}

func ReadLine() string {
	cmd := Cmd{}
	run := true

	for run {
		key := make([]byte, 4)

		os.Stdin.Read(key)
		r, _ := utf8.DecodeRune(key)
		if unicode.IsPrint(r) {
			cmd.text = append(cmd.text, key[0])
			if cmd.index < len(cmd.text) {
				copy(cmd.text[cmd.index+1:], cmd.text[cmd.index:])
				cmd.text[cmd.index] = key[0]
			}
			cmd.index++
		} else {
			k := binary.BigEndian.Uint32(key)
			f, ok := keyMap[k]
			if ok {
				f.(func(*Cmd))(&cmd)
			} /* else {
				fmt.Println("key: ", k)
			} */
			run = k != 167772160
		}

		output := "\r\033[2K"
		output += string(cmd.text)
		for i := cmd.index; i < len(cmd.text); i++ {
			output += "\b"
		}
		fmt.Print(output)
	}
	fmt.Println()

	return string(cmd.text)
}
