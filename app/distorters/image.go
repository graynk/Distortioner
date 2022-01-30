package distorters

import (
	"log"
	"os/exec"

	"github.com/pkg/errors"
)

func DistortImage(path string) error {
	err := exec.Command(
		"mogrify",
		"-scale", "512x512>", // A reasonable cutoff, I hope
		"-liquid-rescale", "50%",
		"-scale", "200%",
		path).Run()
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
	}
	return err
}
