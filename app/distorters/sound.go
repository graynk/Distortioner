package distorters

func DistortSound(filename, output string) error {
	return runFfmpeg(
		"-i", filename,
		"-vn",
		"-c:a", "libopus",
		"-af", "vibrato=f=6:d=1",
		output)
}
