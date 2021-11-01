package main

import (
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func distortVideo(filename, output string, progressChan chan string) {
	progressChan <- "Extracting frames..."
	defer close(progressChan)
	framesFir := filename + "Frames"
	err := os.Mkdir(framesFir, 0755)
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
		return
	}
	defer os.RemoveAll(framesFir)
	frameRateFraction, totalFrames, err := getFrameRateFractionAndFrameCount(filename)
	if err != nil {
		progressChan <- "Failed"
		return
	}
	numberedFileName := fmt.Sprintf("%s/%s%%04d.png", framesFir, filename)
	err = extractFramesFromVideo(frameRateFraction, filename, numberedFileName)
	if err != nil {
		progressChan <- "Failed"
		return
	}

	distortedFrames := 0
	doneChan := make(chan int, 8)
	go poolDistortImages(numberedFileName, totalFrames, doneChan)

	lastUpdate := time.Now()
	for distortedFrames != totalFrames {
		framesDistorted := <-doneChan
		if framesDistorted == -1 {
			progressChan <- "Failed"
			return
		}
		distortedFrames += framesDistorted
		now := time.Now()
		if now.Sub(lastUpdate).Seconds() > 2 {
			lastUpdate = now
			progressChan <- generateProgressMessage(distortedFrames, totalFrames)
		}
	}
	progressChan <- "Collecting frames..."
	err = collectFramesToVideo(numberedFileName, frameRateFraction, output)
	if err != nil {
		progressChan <- "Failed"
	}
	return
}

func getFrameRateFractionAndFrameCount(filename string) (string, int, error) {
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-count_frames",
		"-show_entries", "stream=nb_read_frames,avg_frame_rate",
		filename).Output()
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
		return "", 0, err
	}
	split := strings.Split(string(output), "\n")
	frameCount, err := strconv.Atoi(split[1])
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
	}
	return split[0], frameCount, err
}

func extractFramesFromVideo(frameRateFraction, filename, numberedFileName string) error {
	return runFfmpeg("-i", filename,
		"-r", frameRateFraction,
		numberedFileName)
}

func collectFramesToVideo(numberedFileName, frameRateFraction, filename string) error {
	return runFfmpeg("-r", frameRateFraction,
		"-i", numberedFileName,
		"-f", "mp4",
		"-c:v", "libx264",
		"-an",
		"-pix_fmt", "yuv420p",
		filename)
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
			err := distortImage(fmt.Sprintf(numberedFileName, frame))
			if err != nil {
				doneChan <- -1
			}
		}(frame)
	}
}
