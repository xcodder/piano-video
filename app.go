package main

import (
	"piano-video/videogenerator"
)

func main() {
	const midiFilePath = "sample-midis/minuetg.mid"
	videogenerator.GenerateVideo(midiFilePath)
}
