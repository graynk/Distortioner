package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
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
	frameRateFraction := getFrameRateFraction(filename)
	numberedFileName := fmt.Sprintf("%s/%s%%03d.png", framesFir, filename)
	extractFramesFromVideo(frameRateFraction, filename, numberedFileName)

	files, err := os.ReadDir(framesFir)
	distortedFrames := 0
	totalFrames := len(files)
	doneChan := make(chan int, 8)
	go poolDistortImages(framesFir, files, doneChan)

	lastUpdate := time.Now()
	for distortedFrames != totalFrames {
		distortedFrames += <-doneChan
		now := time.Now()
		if now.Sub(lastUpdate).Seconds() > 1 {
			lastUpdate = now
			progressChan <- generateProgressMessage(distortedFrames, totalFrames)
		}
	}
	progressChan <- "Collecting frames..."
	collectFramesToVideo(numberedFileName, frameRateFraction, output)
	progressChan <- "Done!"
	close(progressChan)
}

func getFrameRateFraction(filename string) string {
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-show_entries", "stream=r_frame_rate",
		filename).Output()
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return string(output)
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
		"-an",
		filename).Run()
	if err != nil {
		log.Println(err)
		panic(err)
	}
}

func poolDistortImages(framesDir string, files []os.DirEntry, doneChan chan int) {
	cpuCount := runtime.NumCPU()
	sem := make(chan bool, cpuCount)
	for _, frame := range files {
		sem <- true
		go func(frame os.DirEntry) {
			defer func() {
				<-sem
				doneChan <- 1
			}()
			distortImage(fmt.Sprintf("%s/%s", framesDir, frame.Name()))
		}(frame)
	}
}
