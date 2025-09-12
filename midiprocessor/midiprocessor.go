package midiprocessor

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
	"sync/atomic"
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

type FallingNote struct {
	Note   int
	Y      float64
	Height float64
}

var w float64 = 1920
var h float64 = 1080
var octavesDisplayed = 7
var whiteKeysShown = 7 * octavesDisplayed
var keyW float64 = (w - 40) / float64(whiteKeysShown)
var keyH float64 = keyW * 6
var bKeyW float64 = keyW / 1.7
var bKeyH float64 = keyH / 1.6
var pressedKeys = map[int]bool{}
var frameToPressedKeys = map[int]map[int]bool{}
var frameAction = map[int]map[int]bool{}
var frameFallingNotes = map[int][]FallingNote{}
var frameBpm = map[int]float64{}
var fps = 60
var keyY = h - keyH
var musicTime float64
var startDelaySec float64 = 3

const DEBUG = false

func updateFrameKeys(actions map[int]bool) {
	for m, v := range actions {
		if v {
			pressedKeys[m] = true
		} else {
			delete(pressedKeys, m)
		}
	}
}

var tickBpm = map[int]float64{}

func setTickBpm(tick int, bpm float64) {
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

	return accumulatedTime + startDelaySec
}

func setFrameAction(frame int, key int, isPressed bool) {
	if _, exists := frameAction[frame]; !exists {
		frameAction[frame] = map[int]bool{}
	}
	frameAction[frame][key] = isPressed
}

func setNoteAction(key int, onTickFrame, offTickFrame int) {
	setFrameAction(onTickFrame, key, true)
	setFrameAction(offTickFrame, key, false)

	var startRainingNoteFrame = onTickFrame - (3 * fps)

	for i := startRainingNoteFrame; i < offTickFrame; i++ {
		var maxRange = keyY
		var rangePerFrame = float64(maxRange) / float64(3*fps)
		var relativeFrame = i - startRainingNoteFrame

		if _, exists := frameFallingNotes[i]; !exists {
			frameFallingNotes[i] = []FallingNote{}
		}

		var noteFullHeight float64 = (float64(offTickFrame) - float64(onTickFrame)) * rangePerFrame
		var noteY = (rangePerFrame * float64(relativeFrame)) - float64(noteFullHeight)
		var noteDisplayedHeight float64
		if noteY+noteFullHeight > maxRange {
			noteDisplayedHeight = maxRange - noteY
		} else {
			noteDisplayedHeight = noteFullHeight
		}

		frameFallingNotes[i] = append(frameFallingNotes[i], FallingNote{
			Note:   key,
			Y:      noteY,
			Height: noteDisplayedHeight,
		})
	}
}

func setFrameBpmChange(frame int, bpm float64) {
	frameBpm[frame] = bpm
}

// 600 3s 180 fps
// 200 1s 60 fps
// 3.3 frame

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
	for i := 0; i < octavesDisplayed; i++ {
		drawKeyboardOctave(dc, i, pressedKeys)
	}
}

func isWhiteNote(note int) bool {
	var keyInOctave = note % 12
	var blackKeys = map[int]bool{1: true, 3: true, 6: true, 8: true, 10: true}
	return !blackKeys[keyInOctave]
}

func countWhiteNotes(note int) int {
	var whiteNotes = 0
	for i := 0; i < note; i++ {
		if isWhiteNote(i) {
			whiteNotes++
		}
	}
	return whiteNotes
}

func getNoteByKeyAndOctave(key, octave int) int {
	return (octave * 12) + key
}
func getNoteXPosition(note int) float64 {
	var isWhite = isWhiteNote(note)
	var lastWhiteNotePosition = float64(countWhiteNotes(note))*keyW + 20

	if isWhite {
		return lastWhiteNotePosition
	}

	return float64(countWhiteNotes(note-1))*keyW + 20 + keyW/1.5

}

func drawFallingNotes(dc *gg.Context, fallingNotes []FallingNote) {
	for _, n := range fallingNotes {
		var whiteNote = isWhiteNote(n.Note)
		var x = getNoteXPosition(n.Note)

		if whiteNote {
			dc.DrawRoundedRectangle(x, n.Y, keyW, n.Height, 4)
			dc.SetRGB(1, 0.5, 0)
		} else {
			dc.DrawRoundedRectangle(x, n.Y, bKeyW, n.Height, 4)
			dc.SetRGB(0.8, 0.3, 0)
		}

		dc.FillPreserve()
		dc.SetRGBA(0, 0, 0, 1)
		dc.SetLineWidth(1)
		dc.Stroke()
	}
}

func drawScreenAxes(dc *gg.Context, frame int) {
	for i := 0; i < octavesDisplayed; i++ {
		var x = getNoteXPosition(getNoteByKeyAndOctave(0, i))
		dc.SetRGBA(1, 1, 1, 0.3)
		dc.SetLineWidth(0.5)
		dc.DrawLine(x, 0, x, h)
		dc.Stroke()

		var x2 = getNoteXPosition(getNoteByKeyAndOctave(5, i))
		dc.SetRGBA(1, 1, 1, 0.1)
		dc.SetLineWidth(0.5)
		dc.DrawLine(x2, 0, x2, h)
		dc.Stroke()
	}
}

func drawCNotesNotation(dc *gg.Context) {
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}

	face := truetype.NewFace(font, &truetype.Options{Size: 16})

	dc.SetFontFace(face)
	for i := 0; i < octavesDisplayed; i++ {
		if i == 3 {
			dc.SetRGBA(0, 0, 0, 0.8)
		} else {
			dc.SetRGBA(0, 0, 0, 0.5)
		}
		var x = getNoteXPosition(getNoteByKeyAndOctave(0, i))
		dc.DrawString(fmt.Sprintf("C%d", i+1), (x + 7), h-10)
	}
}

func prepareScreen(dc *gg.Context) {
	dc.SetRGB(0.17, 0.17, 0.17)
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

func createFrame(dc *gg.Context, i int) {
	var framePressedKeys = frameToPressedKeys[i]
	var frameFallingNotes = frameFallingNotes[i]
	prepareScreen(dc)
	drawScreenAxes(dc, i)
	drawKeyboard(dc, framePressedKeys)
	drawCNotesNotation(dc)
	drawFallingNotes(dc, frameFallingNotes)

	var frStr string
	if i+1 > 9999 {
		frStr = fmt.Sprintf("%d", i+1)
	} else if i+1 > 999 {
		frStr = fmt.Sprintf("0%d", i+1)
	} else if i+1 > 99 {
		frStr = fmt.Sprintf("00%d", i+1)
	} else if i+1 > 9 {
		frStr = fmt.Sprintf("000%d", i+1)
	} else {
		frStr = fmt.Sprintf("0000%d", i+1)
	}

	if DEBUG == true {
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			log.Fatal(err)
		}

		face := truetype.NewFace(font, &truetype.Options{Size: 9})

		dc.SetFontFace(face)
		dc.DrawString(fmt.Sprintf("FRAME %s", frStr), 30, 30)
	}

	dc.SavePNG(fmt.Sprintf("frames/fr%s.png", frStr))
}
func createFrames() {

	const maxWorkers = 50
	sem := make(chan struct{}, maxWorkers)
	contexts := make(chan *gg.Context, maxWorkers)

	var wg sync.WaitGroup

	var totalFrames = fps * int(math.Round(musicTime))
	var finishedFrames atomic.Uint64
	var startTime = time.Now()

	for i := 0; i < maxWorkers; i++ {
		dc := gg.NewContext(int(w), int(h))
		contexts <- dc
	}

	for i := 0; i < totalFrames; i++ {
		wg.Add(1)
		sem <- struct{}{}
		dc := <-contexts
		go func(dc *gg.Context, i int) {
			defer wg.Done()
			createFrame(dc, i)
			f := finishedFrames.Add(1)
			if int(f)%(fps*10) == 0 {
				fmt.Printf("Finished frames: %d/%d\tavg time per frame: %.4f\n", f, totalFrames, time.Since(startTime).Seconds()/float64(f))
			}
			<-sem
			contexts <- dc
		}(dc, i)
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

			setNoteAction(note, int(onTickFrame), int(offTickFrame))
		}

		var trackTimeSeconds = getTickTime(track.Time, quarterNoteTicks)
		if trackTimeSeconds > musicTime {
			musicTime = trackTimeSeconds
		}
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
		fmt.Printf("ffmpeg ")
		for _, v := range cmdArgs {
			fmt.Printf("%s ", v)
		}

		fmt.Printf("Error executing FFmpeg command: %v\n", err)
		return err
	}

	return nil
}

func Generate() {
	// var midiFile = "minuetg.mid"
	// var midiFile = "Dance in E Minor - test_5_min.mid"
	var midiFile = "Hungarian Rhapsody No. 2 in C# Minor.mid"

	removeFrames()

	f, err := os.Open("./" + midiFile)
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

	err = createVideoFromFrames("frames", outputMp3Path, "output/"+midiFile+"."+fmt.Sprintf("%d", fps)+".mp4")
	if err != nil {
		fmt.Println("Error: ffmpeg could not create video")
	}

	fmt.Println("E4 ", time.Since(executionStartTime).Seconds())

	// removeFrames()

	fmt.Println("E5 ", time.Since(executionStartTime).Seconds())

	// removeAudioFile(outputMp3Path)

	executionTime := time.Since(executionStartTime)

	fmt.Printf("Execution time: %f seconds\n", executionTime.Seconds())

}
