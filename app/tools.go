package main

import (
	"fmt"
	"strconv"
	"strings"
)

const progress = "Processing frames...\n<code>[----------] %d%%</code>"

func uniqueFileName(fileId string, timestamp int64) string {
	return fileId + strconv.FormatInt(timestamp, 10)
}

func generateProgressMessage(done, total int) string {
	fraction := float64(done) / float64(total)
	message := fmt.Sprintf(progress, int(fraction*100))
	return strings.Replace(message, "-", "=", int(fraction*10))
}
