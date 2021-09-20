package main

import (
	"log"
	"os/exec"
)

func distortSound(filename, output string) {
	err := exec.Command(
		"ffmpeg",
		"-i", filename,
		"-c:a", "libopus",
		"-af", "vibrato=f=6:d=1",
		output).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}
