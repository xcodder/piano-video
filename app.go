package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"sync"
	"time"
	"videos2/midiparser"

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

var w float64 = 1400
var h float64 = 600
var whiteKeysShown = 7 * 8
var keyW float64 = (w - 40) / float64(whiteKeysShown)
var keyH float64 = keyW * 2.5
var bKeyW float64 = keyW / 1.7
var bKeyH float64 = keyH / 1.6
var pressedKeys = map[int]bool{}
var frameToPressedKeys = map[int]map[int]bool{}
var frameAction = map[int]map[int]bool{}
var frameBpm = map[int]int{}
var fps = 60

var musicTime float64
var lastBpm = 0

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

func setTickBpm(tick int, bpm int) {
	tickBpm[tick] = bpm
}

//
//  3800
// map[int]int {
// 0: 120,
// 3600: 160,
// }

// quarterTicks 100

func getTickTime(tick int, quarterNoteTicks int) float64 {

	var orderedBpmTicks = []int{}

	for bpmTick := range tickBpm {
		orderedBpmTicks = append(orderedBpmTicks, bpmTick)
	}

	sort.Ints(orderedBpmTicks)
	slices.Reverse(orderedBpmTicks)

	currentTick := tick

	var accumulatedTime float64 = 0
	for i := range orderedBpmTicks {
		v := orderedBpmTicks[i]

		if tick > v {
			var bpmTicks = currentTick - v
			currentTick = v
			var tBpm = tickBpm[v]

			var beatTime = 60 / float64(tBpm)
			var onTickTime = float64(bpmTicks) / float64(quarterNoteTicks) * float64(beatTime)
			accumulatedTime += onTickTime
		}
	}
	return accumulatedTime
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

func drawKeyboardKey(dc *gg.Context, x, y float64, isPressed bool) {
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

func drawKeyboardBlackKey(dc *gg.Context, x, y float64, isPressed bool) {
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

func drawKeyboardOctave(dc *gg.Context, octaveI int, pressedKeys map[int]bool) {

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
		drawKeyboardKey(dc, keyX, keyY, isPressed)

		if i == 2 || i == 6 {
			continue
		}
		whiteKeysWithBlackKeys = append(whiteKeysWithBlackKeys, []float64{keyX, keyY, float64(keyNote)})
	}

	for i := 0; i < len(whiteKeysWithBlackKeys); i++ {
		wk := whiteKeysWithBlackKeys[i]
		var isPressed = pressedKeys[int(wk[2])+1]
		drawKeyboardBlackKey(dc, wk[0], wk[1], isPressed)
	}
}

func drawKeyboard(dc *gg.Context, pressedKeys map[int]bool) {
	for i := 0; i < 8; i++ {
		drawKeyboardOctave(dc, i, pressedKeys)
	}
}

func prepareScreen(dc *gg.Context) {
	dc.SetRGB(1, 1, 1)
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()
}

func createFramesKeyboard() {
	for i := 0; i < fps*int(math.Round(musicTime)); i++ {
		var framePressedKeys = map[int]bool{}
		if v, exists := frameAction[i]; exists {
			updateFrameKeys(v)
		}
		for k, v := range pressedKeys {
			framePressedKeys[k] = v
		}

		frameToPressedKeys[i] = framePressedKeys
	}
}

func createFrameGoroutine(i int) func() {
	return func() {
		var dc = gg.NewContext(int(w), int(h))
		createFrame(i, dc)
	}
}
func createFrame(i int, dc *gg.Context) {
	var framePressedKeys = frameToPressedKeys[i]
	prepareScreen(dc)
	drawKeyboard(dc, framePressedKeys)

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

	dc.DrawString(fmt.Sprintf("Time %f", timeNow), 160, 30)
	// dc.DrawString(fmt.Sprintf("Bpm %d", lastBpm), 320, 30)

	dc.SavePNG(fmt.Sprintf("frames/fr%s.png", frStr))
}
func createFrames() {
	var wg sync.WaitGroup

	for i := 0; i < fps*int(math.Round(musicTime)); i++ {
		wg.Go(createFrameGoroutine(i))
	}

	wg.Wait()
}

func removeFrames() {
	var wg sync.WaitGroup
	files, err := filepath.Glob("frames/fr*.png")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		wg.Go(func() {
			if err := os.Remove(f); err != nil {
				log.Fatal(err)
			}
		})
	}

	wg.Wait()
}

func removeAudioFile(filePath string) {
	os.Remove(filePath)
}

func prepareMidi(midiData midiparser.ParsedMidi) {
	var quarterNoteTicks = midiData.Meta.QuarterValue

	var skipChannels = map[byte]bool{}
	for channelId, channel := range midiData.Channels {
		if channel.Patch > 80 {
			skipChannels[channelId] = true
		}
	}
	for _, track := range midiData.Tracks {
		for _, event := range track.Events {
			var note = event.Note
			if note == 0 {
				if event.Meta.Bpm > 0 {
					setTickBpm(event.OnTick, event.Meta.Bpm)
				}
			}
		}
	}

	for _, track := range midiData.Tracks {
		for _, event := range track.Events {
			var note = event.Note
			if note == 0 {
				if event.Meta.Bpm > 0 {
					var onTick = event.OnTick
					var onTickTime = getTickTime(onTick, quarterNoteTicks)
					var onTickFrame = math.Round(onTickTime * float64(fps))
					setFrameBpmChange(int(onTickFrame), event.Meta.Bpm)
				}
			}
		}
	}

	for _, track := range midiData.Tracks {
		for _, event := range track.Events {
			var note = event.Note
			if note == 0 {
				continue
			}

			if _, exists := skipChannels[event.Channel]; exists {
				continue
			}

			note = note - 24
			var onTick = event.OnTick
			var offTick = event.Offtick

			var onTickTime = getTickTime(onTick, quarterNoteTicks)
			var offTickTime = getTickTime(offTick, quarterNoteTicks)

			var onTickFrame = math.Ceil(onTickTime * float64(fps))
			var offTickFrame = math.Floor(offTickTime * float64(fps))

			setFrameAction(int(onTickFrame), note, true)
			setFrameAction(int(offTickFrame), note, false)
		}

		var trackTimeSeconds = getTickTime(track.Time, quarterNoteTicks)
		if trackTimeSeconds > musicTime {
			musicTime = trackTimeSeconds
		}
		musicTime = 30
	}
}
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

	fmt.Println("Midi converted to mp3 successfully!")

	return outputMp3Path
}

func createVideoFromFrames(framesFolder string, audioFilePath string, outputPath string) {

	cmdArgs := []string{
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", framesFolder + "/fr%04d.png",
		"-i", audioFilePath,
		"-preset", "veryfast",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-vcodec", "libx264",
		"-tune", "animation",
		"-y",
		"-t", fmt.Sprintf("%f", musicTime),
		outputPath,
	}

	// for _, v := range cmdArgs {
	// 	fmt.Printf("%s ", v)
	// }

	cmd := exec.Command("ffmpeg", cmdArgs...)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing FFmpeg command: %v\n", err)
		return
	}

	fmt.Println("Video scaled successfully!")
}

func main() {
	// var midiFile = "minuetg.mid"
	// var midiFile = "Dance in E Minor - test_5_min.mid"
	var midiFile = "Bach JS Toccata Fuge D Minor.mid"

	removeFrames()

	f, err := os.Open(midiFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	executionStartTime := time.Now()
	parsedMidi, err := midiparser.ParseFile(f)
	if err != nil {
		panic(err)
	}

	rankingsJSON, _ := json.Marshal(parsedMidi)
	os.WriteFile("./midioutput.json", rankingsJSON, 0644)

	fmt.Println("E1 ", time.Since(executionStartTime).Seconds())
	prepareMidi(parsedMidi)

	createFramesKeyboard()

	fmt.Println("E2 ", time.Since(executionStartTime).Seconds())
	createFrames()

	outputMp3Path := convertMidiToMp3(midiFile)
	fmt.Println("E3 ", time.Since(executionStartTime).Seconds())

	createVideoFromFrames("frames", outputMp3Path, "output/"+midiFile+"."+fmt.Sprintf("%d", fps)+".mp4")

	fmt.Println("E4 ", time.Since(executionStartTime).Seconds())

	// removeFrames()

	fmt.Println("E5 ", time.Since(executionStartTime).Seconds())

	removeAudioFile(outputMp3Path)

	executionTime := time.Since(executionStartTime)

	fmt.Printf("Execution time: %f seconds\n", executionTime.Seconds())

}
