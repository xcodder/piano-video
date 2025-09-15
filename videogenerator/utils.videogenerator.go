package videogenerator

import "path/filepath"

func getFileNameWithoutExtension(filePath string) string {
	fileName := filepath.Base(filePath)
	return fileName[:len(fileName)-len(filepath.Ext(fileName))]
}
