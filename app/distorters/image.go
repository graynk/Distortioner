package distorters

import (
	"log"

	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v3/imagick"
)

func DistortImage(path string) error {
	_, err := imagick.ConvertImageCommand([]string{
		"convert",
		"-scale", "512x512>", // A reasonable cutoff, I hope
		"-liquid-rescale", "50%",
		"-scale", "200%",
		path, path})
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
	}
	return err
}
