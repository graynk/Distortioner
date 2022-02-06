package distorters

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/pkg/errors"
)

func DistortVideoSticker(filename, output string, group *sync.WaitGroup) {
	defer group.Done()
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
		return
	} else if duration > 30 {
		return
	}
	numberedFileName := fmt.Sprintf("%s/%s%%04d.png", framesDir, filename)
	err = extractFramesFromVideoSticker(frameRateFraction, filename, numberedFileName)
	if err != nil {
		return
	}

	distortedFrames := 0
	doneChan := make(chan int, 8)
	go poolDistortImages(framesDir, doneChan)

	for totalFrames := <-doneChan; distortedFrames != totalFrames; {
		framesDistorted := <-doneChan
		if framesDistorted == -1 {
			return
		}
		distortedFrames += framesDistorted
	}
	err = collectFramesToVideoSticker(numberedFileName, frameRateFraction, output)
	if err != nil {
		log.Println(err)
	}
	return
}
