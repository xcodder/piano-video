package videogenerator

import (
	"fmt"
	"path/filepath"
)

func getFileNameWithoutExtension(filePath string) string {
	fileName := filepath.Base(filePath)
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}

func getOutputVideoPath(midiFilePath string) string {
	return fmt.Sprintf("%s/%s.mp4", outputFolderPath, getFileNameWithoutExtension(midiFilePath))
}
