package videogenerator

import (
	"fmt"
	"os/exec"
)

func createVideoFromFrames(framesFolder string, audioFilePath string, outputPath string) error {

	cmdArgs := []string{
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", framesFolder + "/fr%05d.png",
		"-itsoffset", fmt.Sprintf("%fs", startDelaySec),
		"-i", audioFilePath,
		"-map", "0:v", "-map", "1:a",
		"-preset", "veryfast",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-vcodec", "libx264",
		"-tune", "animation",
		"-y",
		"-t", fmt.Sprintf("%f", musicTime),
		outputPath,
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)

	if err := cmd.Run(); err != nil {
		var fullCmd string
		fullCmd += "ffmpeg "
		for _, v := range cmdArgs {
			fullCmd += fmt.Sprintf("%s ", v)
		}

		return fmt.Errorf("error executing FFmpeg command: %s; %v", fullCmd, err)
	}

	return nil
}
