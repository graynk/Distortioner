package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func distortVideo(filename, output string, progressChan chan string) {
	progressChan <- "Extracting frames..."
	framesFir := filename + "Frames"
	err := os.Mkdir(framesFir, 0755)
	if err != nil {
		log.Println(err)
		panic(err)
		return
	}
	defer os.RemoveAll(framesFir)
	frameRateFraction, totalFrames := getFrameRateFractionAndFrameCount(filename)
	numberedFileName := fmt.Sprintf("%s/%s%%03d.png", framesFir, filename)
	extractFramesFromVideo(frameRateFraction, filename, numberedFileName)

	distortedFrames := 0
	doneChan := make(chan int, 8)
	go poolDistortImages(numberedFileName, totalFrames, doneChan)

	for distortedFrames != totalFrames {
		distortedFrames += <-doneChan
		progressChan <- generateProgressMessage(distortedFrames, totalFrames)
	}
	progressChan <- "Collecting frames..."
	collectFramesToVideo(numberedFileName, frameRateFraction, output)
	progressChan <- "Done!"
	close(progressChan)
}

func getFrameRateFractionAndFrameCount(filename string) (string, int) {
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-count_packets",
		"-show_entries", "stream=nb_read_packets,r_frame_rate",
		filename).Output()
	if err != nil {
		log.Println(err)
		panic(err)
	}
	split := strings.Split(string(output), "\n")
	frameCount, err := strconv.Atoi(split[1])
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return split[0], frameCount
}

func extractFramesFromVideo(frameRateFraction, filename, numberedFileName string) {
	err := exec.Command(
		"ffmpeg",
		"-i", filename,
		"-r", frameRateFraction,
		numberedFileName).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func collectFramesToVideo(numberedFileName, frameRateFraction, filename string) {
	err := exec.Command("ffmpeg",
		"-i", numberedFileName,
		"-r", frameRateFraction,
		"-f", "mp4",
		"-c:v", "libx264",
		filename).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func poolDistortImages(numberedFileName string, frameCount int, doneChan chan int) {
	cpuCount := runtime.NumCPU()
	sem := make(chan bool, cpuCount)
	for frame := 1; frame <= frameCount; frame++ {
		sem <- true
		go func(frame int) {
			defer func() {
				<-sem
				doneChan <- 1
			}()
			distortImage(fmt.Sprintf(numberedFileName, frame))
		}(frame)
	}
}
