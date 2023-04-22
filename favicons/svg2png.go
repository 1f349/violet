package favicons

import (
	"bytes"
	"fmt"
	"os/exec"
)

func svg2png(inkscapeCmd string, in []byte) (out []byte, err error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command(inkscapeCmd, "--export-type", "png", "--export-filename", "-", "--export-background-opacity", "0", "--pipe")
	cmd.Stdin = bytes.NewBuffer(in)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if e := cmd.Run(); e != nil {
		err = fmt.Errorf("%s\nSTDERR:\n%s", e.Error(), stderr.String())
		return
	}

	if stdout.Len() == 0 {
		err = fmt.Errorf("got no data from inkscape")
		return
	}

	out = stdout.Bytes()
	return
}
