package videogenerator

var blackKeysInOctave = map[int]bool{1: true, 3: true, 6: true, 8: true, 10: true}
var pressedKeys = map[int]PlayingNote{}
var frameToPressedKeys = map[int]map[int]PlayingNote{}
var frameAction = map[int]map[int]PlayingNote{}
var frameFallingNotes = map[int][]FallingNote{}
var frameBpm = map[int]float64{}
var tickBpm = map[int]float64{}
var keyY = h - keyH
var musicTime float64

var colorOrange = Color{1, 0.5, 0}
var colorGreen = Color{0.2, 1, 0.2}
var colorBlue = Color{0.5, 0.85, 1}
var colorYellow = Color{0.8, 0.6, 0.05}
var colorGrey = Color{0.5, 0.5, 0.5}
var colorPink = Color{1, 0.6, 0.7}

var resolution1080p = ScreenResolution{1920, 1080}
var resolution720p = ScreenResolution{1280, 720}
var resolution480p = ScreenResolution{854, 480}
var resolution360p = ScreenResolution{640, 360}
