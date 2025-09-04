package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"

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
var pressedKeys = map[int]bool{}
var frameAction = map[int]map[int]bool{}
var frameBpm = map[int]int{}
var fps = 24
var musicTime float64
var quarterTicks int

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

var tickBpm = map[int]int{}

func getBpm(tick int) int {
	var lastBpmTick = 0
	for bpmTick := range tickBpm {
		if tick >= bpmTick && bpmTick >= lastBpmTick {
			lastBpmTick = bpmTick
		}
	}
	return tickBpm[lastBpmTick]
}

func setTickBpm(tick int, bpm int) {
	tickBpm[tick] = bpm
}

func setFrameAction(frame int, key int, isPressed bool) {
	if _, exists := frameAction[frame]; !exists {
		frameAction[frame] = map[int]bool{}
	}
	frameAction[frame][key] = isPressed
}

func setFrameBpmChange(frame int, bpm int) {
	frameBpm[frame] = bpm
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
	var lastBpm = 0
	musicTime = 40
	for i := 0; i < fps*int(math.Round(musicTime)); i++ {
		if v, exists := frameAction[i]; exists {
			updateFrameKeys(v)
		}

		prepareScreen()
		drawKeyboard()

		var frStr string
		if i+1 > 999 {
			frStr = fmt.Sprintf("%d", i+1)
		} else if i+1 > 99 {
			frStr = fmt.Sprintf("0%d", i+1)
		} else if i+1 > 9 {
			frStr = fmt.Sprintf("00%d", i+1)
		} else {
			frStr = fmt.Sprintf("000%d", i+1)
		}
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			log.Fatal(err)
		}

		face := truetype.NewFace(font, &truetype.Options{Size: 9})

		dc.SetFontFace(face)
		dc.DrawString(fmt.Sprintf("FRAME %s", frStr), 30, 30)

		// var onTickTime = float64(onTick) / float64(quarterTicks) * float64(beatTime)

		var timeNow = float64(i) / float64(fps)

		if bpm := frameBpm[i]; bpm > 0 {
			lastBpm = bpm
		}

		dc.DrawString(fmt.Sprintf("Time %f", timeNow), 160, 30)
		dc.DrawString(fmt.Sprintf("Bpm %d", lastBpm), 320, 30)
		dc.SavePNG(fmt.Sprintf("frames/fr%s.png", frStr))

		dc.Clear()
	}
}

func removeFrames() {
	files, err := filepath.Glob("frames/fr*.png")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}
}

func prepareMidi(midiData MidiData) {
	var quarterNoteTicks = midiData.Meta.QuarterValue
	quarterTicks = quarterNoteTicks

	for _, track := range midiData.Tracks {
		for _, event := range track.Events {
			var note = event.Note
			if note == 0 {
				if event.Meta.Bpm > 0 {
					setTickBpm(event.OnTick, event.Meta.Bpm)
					var onTick = event.OnTick
					var bpm = getBpm(onTick)
					var beatTime = 60 / float64(bpm)
					var onTickTime = float64(onTick) / float64(quarterNoteTicks) * float64(beatTime)
					var onTickFrame = math.Floor(onTickTime * float64(fps))
					setFrameBpmChange(int(onTickFrame), event.Meta.Bpm)
				}
				continue
			}
			note = note - 24
			var onTick = event.OnTick
			var offTick = event.Offtick

			var bpm = getBpm(onTick)
			var beatTime = 60 / float64(bpm)
			var onTickTime = float64(onTick) / float64(quarterNoteTicks) * float64(beatTime)
			var offTickTime = float64(offTick) / float64(quarterNoteTicks) * float64(beatTime)

			var onTickFrame = math.Floor(onTickTime * float64(fps))
			var offTickFrame = math.Floor(offTickTime * float64(fps))

			setFrameAction(int(onTickFrame), note, true)
			setFrameAction(int(offTickFrame), note, false)
		}

		// var trackTimeSeconds = float64(track.Time) / float64(quarterNoteTicks) * float64(beatTime)
		// if trackTimeSeconds > musicTime {
		// 	musicTime = trackTimeSeconds
		// }
	}
}
func convertMidiToMp3(midiFilePath string) string {
	var outputMp3Path = midiFilePath + ".mp3"
	timidityCmdArgs := []string{
		midiFilePath, "-Ow",
		"-o", outputMp3Path,
	}

	cmd := exec.Command("timidity", timidityCmdArgs...)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing timidity command: %v\n", err)
		return ""
	}

	fmt.Println("Midi converted to mp3 successfully!")

	return outputMp3Path
}

func createVideoFromFrames(framesFolder string, audioFilePath string) {

	cmdArgs := []string{
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", framesFolder + "/fr%04d.png",
		"-i", audioFilePath,
		"-y",
		"video2.mp4",
	}

	cmd := exec.Command("ffmpeg", cmdArgs...)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing FFmpeg command: %v\n", err)
		return
	}

	fmt.Println("Video scaled successfully!")
}

func main() {
	// var midiFile = "minuetg.mid"
	var midiFile = "moonlight-sonata.mid"

	jsonFile, err := os.Open("midi.json")
	if err != nil {
		log.Fatal(err)
	}

	var midiData MidiData
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &midiData)

	prepareMidi(midiData)

	createFrames()

	outputMp3Path := convertMidiToMp3(midiFile)

	createVideoFromFrames("frames", outputMp3Path)

	removeFrames()

}
