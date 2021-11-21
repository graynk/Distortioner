package main

func collectAnimationAndSound(animation, sound, output string) error {
	if sound != "" {
		return runFfmpeg("-i", animation,
			"-i", sound,
			"-c:v", "copy",
			"-c:a", "copy",
			output)
	}
	return runFfmpeg("-i", animation,
		"-c:v", "copy",
		"-an",
		output)
}
