package distorters

func DistortSound(filename string) (string, error) {
	output := filename + ".ogg"
	return output, runFfmpeg(
		"-i", filename,
		"-vn",
		"-c:a", "libopus",
		"-af", "vibrato=f=6:d=1",
		output)
}
