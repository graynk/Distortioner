package distorters

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/graynk/distortioner/tools"
)

const (
	Failed  = "Failed"
	TooLong = "Senpai, it's too long.."
	TooBig  = "Senpai, it's too big.."
)

func DistortVideo(filename, output string, progressChan chan string) {
	progressChan <- "Extracting frames..."
	defer close(progressChan)
	framesDir := filename + "Frames"
	err := os.Mkdir(framesDir, 0755)
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
		return
	}
	defer os.RemoveAll(framesDir)
	frameRateFraction, duration, err := GetFrameRateFractionAndDuration(filename)
	if err != nil {
		progressChan <- Failed
		return
	} else if duration > 30 {
		progressChan <- TooLong
		return
	}
	numberedFileName := fmt.Sprintf("%s/%s%%04d.png", framesDir, filename)
	err = extractFramesFromVideo(frameRateFraction, filename, numberedFileName)
	if err != nil {
		progressChan <- Failed
		return
	}

	distortedFrames := 0
	doneChan := make(chan int, 8)
	go poolDistortImages(framesDir, doneChan)

	lastUpdate := time.Now()
	for totalFrames := <-doneChan; distortedFrames != totalFrames; {
		framesDistorted := <-doneChan
		if framesDistorted == -1 {
			progressChan <- Failed
			return
		}
		distortedFrames += framesDistorted
		now := time.Now()
		if now.Sub(lastUpdate).Seconds() > 2 {
			lastUpdate = now
			progressChan <- tools.GenerateProgressMessage(distortedFrames, totalFrames)
		}
	}
	progressChan <- "Collecting frames..."
	err = collectFramesToVideo(numberedFileName, frameRateFraction, output)
	if err != nil {
		progressChan <- Failed
	}
	return
}

func GetFrameRateFractionAndDuration(filename string) (string, float64, error) {
	output, err := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v",
		"-of", "default=noprint_wrappers=1:nokey=1",
		"-show_entries", "stream=avg_frame_rate",
		"-show_entries", "format=duration",
		filename).Output()
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
		return "", 0, err
	}
	split := strings.Split(string(output), "\n")
	duration, err := strconv.ParseFloat(split[1], 32)
	if err != nil {
		err = errors.WithStack(err)
		log.Println(err)
	}
	return split[0], duration, err
}

func extractFramesFromVideo(frameRateFraction, filename, numberedFileName string) error {
	return runFfmpeg("-i", filename,
		"-r", frameRateFraction,
		numberedFileName)
}

func extractFramesFromVideoSticker(frameRateFraction, filename, numberedFileName string) error {
	return runFfmpeg("-vcodec", "libvpx-vp9",
		"-i", filename,
		"-r", frameRateFraction,
		"-pix_fmt", "rgba",
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

func collectFramesToVideoSticker(numberedFileName, frameRateFraction, filename string) error {
	return runFfmpeg("-r", frameRateFraction,
		"-i", numberedFileName,
		"-f", "webm",
		"-c:v", "libvpx-vp9",
		"-b:v", "85k", // ebuchiy shakal mode activated
		"-an",
		"-pix_fmt", "yuva420p",
		filename)
}

func poolDistortImages(frameDir string, doneChan chan int) {
	cpuCount := runtime.NumCPU()
	sem := make(chan bool, cpuCount)
	frames, err := os.ReadDir(frameDir)
	if err != nil {
		doneChan <- -1
		doneChan <- -1
		return
	}
	doneChan <- len(frames)
	for i, frame := range frames {
		sem <- true
		go func(i int, frame string) {
			defer func() {
				<-sem
				doneChan <- 1
			}()
			err := DistortImage(fmt.Sprintf("%s/%s", frameDir, frame))
			if err != nil {
				doneChan <- -1
			}
		}(i, frame.Name())
	}
}
