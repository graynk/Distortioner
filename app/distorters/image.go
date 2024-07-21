package distorters

import (
	"log"
	"os/exec"
	"syscall"

	"github.com/pkg/errors"
)

func DistortImage(path string) error {
	cmd := exec.Command(
		"magick",
		path,
		"-resize", "512x512>", // A reasonable cutoff, I hope
		"-liquid-rescale", "50%",
		"-resize", "200%",
		path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err := cmd.Run()
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
	}
	return err
}
