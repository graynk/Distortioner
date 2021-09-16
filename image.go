package main

import (
	"log"
	"os/exec"
)

func distortImage(path string) {
	err := exec.Command(
		"mogrify",
		"-scale", "300x300>",
		"-liquid-rescale", "50%",
		"-scale", "200%",
		path).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}
