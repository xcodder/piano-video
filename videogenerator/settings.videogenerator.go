package videogenerator

import "fmt"

var colors = []Color{colorOrange, colorGreen, colorBlue, colorGrey, colorGrey}

var defaultResolution = resolution1080p

var w float64 = defaultResolution[0]
var h float64 = defaultResolution[1]
var octavesDisplayed = 7

var whiteKeysDisplayed = 7 * octavesDisplayed
var keyW float64 = (w - 40) / float64(whiteKeysDisplayed)
var keyH float64 = keyW * 6
var bKeyW float64 = keyW / 1.7
var bKeyH float64 = keyH / 1.6

const DEBUG = false
const fps = 60
const startDelaySec float64 = 3
const fallingNoteBorderRadius float64 = 6
const framesFolderPath = "_frames"
const outputFolderPath = "output"
const midiFilePath = "sample-midis/minuetg.mid"

var outputVideoPath = fmt.Sprintf("%s/%s.mp4", outputFolderPath, getFileNameWithoutExtension(midiFilePath))
