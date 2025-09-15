package videogenerator

import (
	"fmt"
	"os"
	"os/exec"
)

func convertMidiToMp3(midiFilePath string) string {
	var outputMp3Path = midiFilePath + ".wav"
	timidityCmdArgs := []string{
		midiFilePath, "-Ow",
		"--preserve-silence",
		"-o", outputMp3Path,
	}

	cmd := exec.Command("timidity", timidityCmdArgs...)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing timidity command: %v\n", err)
		return ""
	}

	return outputMp3Path
}

func removeAudioFile(filePath string) {
	os.Remove(filePath)
}
