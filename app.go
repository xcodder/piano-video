package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/fogleman/gg"
)

type Track struct {
	Events []Event
	Time   int
}

type Channel struct {
	Name       string `json:"name"`
	Instrument string `json:"instrument"`
	Patch      byte   `json:"patch"`
}

type Meta struct {
	Bpm int `json:"bpm"`
}

type Event struct {
	Note    int  `json:"note"`
	OnTick  int  `json:"on_tick"`
	Offtick int  `json:"off_tick"`
	Channel byte `json:"channel"`
	Meta    Meta `json:"meta"`
}

type HeaderMeta struct {
	QuarterValue int `json:"quarterValue"`
}

type MidiData struct {
	Tracks   []Track          `json:"tracks"`
	Channels map[byte]Channel `json:"channels"`
	Meta     HeaderMeta       `json:"meta"`
}

var w float64 = 1200
var h float64 = 600
var keyW float64 = 25
var keyH float64 = 60
var bKeyW float64 = 25 / 1.7
var bKeyH float64 = keyH / 1.6
var currentFrame = 0
var pressedKeys = map[int]bool{}

var dc = gg.NewContext(int(w), int(h))

func changePressedKeys(addPressedKeys []int, unpressKeys []int) {
	for i := 0; i < len(addPressedKeys); i++ {
		pressedKeys[addPressedKeys[i]] = true
	}
	for i := 0; i < len(unpressKeys); i++ {
		delete(pressedKeys, unpressKeys[i])
	}
}

func updateFrameKeys(actions map[int]bool) {
	for m, v := range actions {
		if v {
			pressedKeys[m] = true
		} else {
			delete(pressedKeys, m)
		}
	}
}

var frameAction = map[int]map[int]bool{}
var fps = 24

func setFrameAction(frame int, key int, isPressed bool) {
	if _, exists := frameAction[frame]; !exists {
		frameAction[frame] = map[int]bool{}
	}
	frameAction[frame][key] = isPressed
}

func setSecondAction(sec int, key int, isPressed bool) {
	var frame = fps * sec
	setFrameAction(frame, key, isPressed)
}

func drawKeyboardKey(x, y float64, isPressed bool) {
	dc.DrawRectangle(x, y, keyW, keyH)

	if isPressed {
		dc.SetRGB(1, 0.5, 0)
	} else {
		dc.SetRGB(1, 1, 1)
	}
	dc.FillPreserve()
	dc.SetRGBA(0, 0, 0, 1)
	dc.SetLineWidth(1)
	dc.Stroke()

}

func drawKeyboardBlackKey(x, y float64, isPressed bool) {
	x = x + keyW/1.5
	dc.DrawRectangle(x, y, bKeyW, bKeyH)

	if isPressed {
		dc.SetRGB(0.8, 0.3, 0)
	} else {
		dc.SetRGB(0.13, 0.13, 0.13)
	}

	dc.FillPreserve()
	dc.SetRGBA(0, 0, 0, 1)
	dc.SetLineWidth(1)
	dc.Stroke()

}

func drawKeyboardOctave(octaveI int) {

	var whiteKeysWithBlackKeys = [][]float64{}
	var octaveX = float64(octaveI)*keyW*7 + 20
	var keyId = 0
	for i := 0; i < 7; i++ {
		if i > 0 {
			if i != 3 {
				keyId = keyId + 2
			} else {
				keyId++
			}
		}

		keyNote := octaveI*12 + keyId
		keyX := octaveX + keyW*float64(i)
		keyY := h - keyH - 20
		var isPressed = pressedKeys[keyNote]
		drawKeyboardKey(keyX, keyY, isPressed)

		if i == 2 || i == 6 {
			continue
		}
		whiteKeysWithBlackKeys = append(whiteKeysWithBlackKeys, []float64{keyX, keyY, float64(keyNote)})
	}

	for i := 0; i < len(whiteKeysWithBlackKeys); i++ {
		wk := whiteKeysWithBlackKeys[i]
		var isPressed = pressedKeys[int(wk[2])+1]
		drawKeyboardBlackKey(wk[0], wk[1], isPressed)
	}
}

func drawKeyboard() {
	for i := 0; i < 6; i++ {
		drawKeyboardOctave(i)
	}
}

func prepareScreen() {
	dc.SetRGB(1, 1, 1)
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()
}
func createFrames() {
	for i := 0; i < fps*5; i++ {
		if v, exists := frameAction[i]; exists {
			updateFrameKeys(v)
		}

		prepareScreen()
		drawKeyboard()

		var frStr string
		if i+1 > 99 {
			frStr = fmt.Sprintf("%d", i+1)
		} else if i+1 > 9 {
			frStr = fmt.Sprintf("0%d", i+1)
		} else {
			frStr = fmt.Sprintf("00%d", i+1)
		}
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			log.Fatal(err)
		}

		face := truetype.NewFace(font, &truetype.Options{Size: 9})

		dc.SetFontFace(face)
		dc.DrawString(fmt.Sprintf("FRAME %s", frStr), 30, 30)
		dc.SavePNG(fmt.Sprintf("frames/fr%s.png", frStr))

		dc.Clear()
	}
}

func prepareMidi(midiData MidiData) {
	var bpm = 60
	var quarterTimeSeconds = 60 / bpm
	for _, track := range midiData.Tracks {
		for _, event := range track.Events {
			var quarterNoteTicks = 384
			var note = event.Note
			var onTick = event.OnTick
			var offTick = event.Offtick

			onTickFrame = fps * onTick

		}
	}
}

func main() {

	jsonFile, err := os.Open("midi.json")
	if err != nil {
		log.Fatal(err)
	}

	var midiData MidiData
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &midiData)

	prepareMidi(midiData)

	setSecondAction(1, 10, true)
	setSecondAction(2, 12, true)
	setSecondAction(3, 14, true)
	setSecondAction(3, 32, true)
	setSecondAction(3, 12, false)
	setSecondAction(4, 16, true)
	setSecondAction(4, 10, false)

	createFrames()
}
