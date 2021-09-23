package main

import (
	"log"
	"os/exec"
)

func collectAnimationAndSound(animation, sound, output string) {
	err := exec.Command(
		"ffmpeg",
		"-i", animation,
		"-i", sound,
		"-c:v", "copy",
		"-c:a", "copy",
		output).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}
