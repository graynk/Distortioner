package main

import (
	"bytes"
	"github.com/pkg/errors"
	"log"
	"os/exec"
)

func runFfmpeg(args ...string) error {
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(
		"ffmpeg",
		args...)
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
