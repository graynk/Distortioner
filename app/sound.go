package main

import (
	"bytes"
	"log"
	"os/exec"
)

func distortSound(filename, output string) error {
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(
		"ffmpeg",
		"-i", filename,
		"-vn",
		"-c:a", "libopus",
		"-af", "vibrato=f=6:d=1",
		output)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		log.Println(outbuf.String())
		log.Println(errbuf.String())
		log.Println(err)
	}
	return err
}
