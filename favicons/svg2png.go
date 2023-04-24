package favicons

import (
	"bytes"
	"fmt"
	"os/exec"
)

// svg2png takes an input inkscape binary path and svg image bytes and outputs
// the png image bytes or an error.
func svg2png(inkscapeCmd string, in []byte) (out []byte, err error) {
	// create stdout and stderr buffers
	var stdout, stderr bytes.Buffer

	// prepare inkscape command and attach buffers
	cmd := exec.Command(inkscapeCmd, "--export-type", "png", "--export-filename", "-", "--export-background-opacity", "0", "--pipe")
	cmd.Stdin = bytes.NewBuffer(in)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// run the command and return errors
	if e := cmd.Run(); e != nil {
		err = fmt.Errorf("%s\nSTDERR:\n%s", e.Error(), stderr.String())
		return
	}

	// error if there is no output
	if stdout.Len() == 0 {
		err = fmt.Errorf("got no data from inkscape")
		return
	}

	// return the raw bytes
	out = stdout.Bytes()
	return
}
