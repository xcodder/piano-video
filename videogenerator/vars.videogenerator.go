package videogenerator

const DEBUG = false

var orangeColor = Color{1, 0.5, 0}
var greenColor = Color{0.2, 1, 0.2}
var blueColor = Color{0.5, 0.85, 1}
var yellowColor = Color{0.8, 0.6, 0.05}
var greyColor = Color{0.5, 0.5, 0.5}
var pinkColor = Color{1, 0.6, 0.7}

var colors = []Color{orangeColor, greenColor, blueColor, greyColor, greyColor}

var resolution1080p = ScreenResolution{1920, 1080}
var resolution720p = ScreenResolution{1280, 720}
var resolution480p = ScreenResolution{854, 480}
var resolution360p = ScreenResolution{640, 360}

var defaultResolution = resolution720p

var w float64 = defaultResolution[0]
var h float64 = defaultResolution[1]
var octavesDisplayed = 7

var whiteKeysDisplayed = 7 * octavesDisplayed
var keyW float64 = (w - 40) / float64(whiteKeysDisplayed)
var keyH float64 = keyW * 6
var bKeyW float64 = keyW / 1.7
var bKeyH float64 = keyH / 1.6
var pressedKeys = map[int]PlayingNote{}
var frameToPressedKeys = map[int]map[int]PlayingNote{}
var frameAction = map[int]map[int]PlayingNote{}
var frameFallingNotes = map[int][]FallingNote{}
var frameBpm = map[int]float64{}
var fps = 60
var keyY = h - keyH
var musicTime float64
var startDelaySec float64 = 3
