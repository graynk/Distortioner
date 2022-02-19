package distorters

import (
	"bytes"
	"log"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

func runFfmpeg(args ...string) error {
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(
		"ffmpeg",
		args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		log.Println(outbuf.String())
		log.Println(errbuf.String())
		err = errors.WithStack(err)
		log.Println(err)
	}
	return err
}
