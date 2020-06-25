package term

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"unicode"
	"unicode/utf8"
	"container/list"
)

type History struct {
	history List
	index int
	length int
}

type Internal struct  {
	history History
}

const internal Internal = {

}

func Init() {
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	// do not display entered characters on the screen
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	// history
	history = List.New()
}

func arrow_up(cmd: []byte, cmd_i: int, cmd_len: int) ([]byte, int, int) {
	byte.Copy(cmd, history[-1])
	cmd_i = cmd_len = len(cmd)
}


const keyMap := map[uint32]func {
	458965248: arrow_up,
	//458965504: arrow_down,
}

func ReadLine() string {
	var cmd []byte = make([]byte, 100)
	var cmd_i = 0
	var cmd_len = 0
	var key []byte = make([]byte, 4)

	for {
		os.Stdin.Read(key)
		r, _ := utf8.DecodeRune(key)
		if unicode.IsPrint(r) {
			fmt.Print(string(r))
			cmd[cmd_i] = key[0]
			cmd_len++
		} else {
			k := binary.BigEndian.Uint32(key)
			fmt.Println("key: ", k)
		}

	}

	return string(cmd)
}
