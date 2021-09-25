package main

import (
	"bytes"
	"log"
	"os/exec"
)

func collectAnimationAndSound(animation, sound, output string) {
	var cmd *exec.Cmd
	var outbuf, errbuf bytes.Buffer
	if sound != "" {
		cmd = exec.Command(
			"ffmpeg",
			"-i", animation,
			"-i", sound,
			"-c:v", "copy",
			"-c:a", "copy",
			output)
	} else {
		cmd = exec.Command(
			"ffmpeg",
			"-i", animation,
			"-c:v", "copy",
			"-an",
			output)
	}
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err := cmd.Run()
	if err != nil {
		log.Println(outbuf.String())
		log.Println(errbuf.String())
		log.Println(err)
		panic(err)
	}
}
