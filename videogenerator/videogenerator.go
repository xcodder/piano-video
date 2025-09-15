package videogenerator

import (
	"fmt"
	"log"
	"math"
	"os"
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

func getDarkerShade(c Color) Color {
	var d = 0.8
	return Color{c.R * d, c.G * d, c.B * d}
}

func setRGBColor(dc *gg.Context, c Color) {
	dc.SetRGB(c.R, c.G, c.B)
}

func getColor(i int) Color {
	return colors[i%len(colors)]
}

func updateFrameKeys(actions map[int]PlayingNote) {
	for m, v := range actions {
		if v.Active {
			pressedKeys[m] = v
		} else {
			delete(pressedKeys, m)
		}
	}
}

func setTickBpm(tick int, bpm float64) {
	tickBpm[tick] = bpm
}

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

func setFrameAction(frame int, key int, isPressed bool, trackIndex int) {
	if _, exists := frameAction[frame]; !exists {
		frameAction[frame] = map[int]PlayingNote{}
	}
	frameAction[frame][key] = PlayingNote{Active: isPressed, Track: trackIndex}
}

func setNoteAction(key, onTickFrame, offTickFrame, trackIndex int) {
	setFrameAction(onTickFrame, key, true, trackIndex)
	setFrameAction(offTickFrame, key, false, trackIndex)

	var startRainingNoteFrame = int(float64(onTickFrame) - (startDelaySec * float64(fps)))

	for i := startRainingNoteFrame; i < offTickFrame; i++ {
		var maxRange = keyY
		var rangePerFrame = float64(maxRange) / float64(startDelaySec*float64(fps))
		var relativeFrame = i - startRainingNoteFrame

		if _, exists := frameFallingNotes[i]; !exists {
			frameFallingNotes[i] = []FallingNote{}
		}

		var minDisplayedHeight = h * 0.0208
		var noteFullHeight float64 = (float64(offTickFrame) - float64(onTickFrame)) * rangePerFrame
		if noteFullHeight < minDisplayedHeight {
			noteFullHeight = minDisplayedHeight
		}
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
			Track:  trackIndex,
		})
	}
}

func setFrameBpmChange(frame int, bpm float64) {
	frameBpm[frame] = bpm
}

func drawKeyboardKey(dc *gg.Context, x, y float64, n PlayingNote) {
	dc.DrawRectangle(x, y, keyW, keyH)

	if n.Active {
		setRGBColor(dc, getColor(n.Track))
	} else {
		dc.SetRGB(1, 1, 1)
	}
	dc.FillPreserve()
	dc.SetRGBA(0, 0, 0, 1)
	dc.SetLineWidth(1)

	dc.Stroke()

}

func drawKeyboardBlackKey(dc *gg.Context, x, y float64, n PlayingNote) {
	x = x + keyW/1.5
	dc.DrawRectangle(x, y, bKeyW, bKeyH)

	if n.Active {
		setRGBColor(dc, getDarkerShade(getColor(n.Track)))
	} else {
		dc.SetRGB(0.13, 0.13, 0.13)
	}

	dc.FillPreserve()
	dc.SetRGBA(0, 0, 0, 1)
	dc.SetLineWidth(1)
	dc.Stroke()
}

func drawKeyboardOctave(dc *gg.Context, octaveI int, pressedKeys map[int]PlayingNote) {

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

func drawKeyboard(dc *gg.Context, pressedKeys map[int]PlayingNote) {
	for i := 0; i < octavesDisplayed; i++ {
		drawKeyboardOctave(dc, i, pressedKeys)
	}
}

func isWhiteNote(note int) bool {
	var keyInOctave = note % 12
	return !blackKeysInOctave[keyInOctave]
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
			dc.DrawRoundedRectangle(x, n.Y, keyW, n.Height, fallingNoteBorderRadius)
			setRGBColor(dc, getColor(n.Track))
		} else {
			dc.DrawRoundedRectangle(x, n.Y, bKeyW, n.Height, fallingNoteBorderRadius)
			setRGBColor(dc, getDarkerShade(getColor(n.Track)))
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

	face := truetype.NewFace(font, &truetype.Options{Size: keyW / 2})

	dc.SetFontFace(face)
	for i := 0; i < octavesDisplayed; i++ {
		if i == 3 {
			dc.SetRGBA(0, 0, 0, 0.8)
		} else {
			dc.SetRGBA(0, 0, 0, 0.5)
		}
		var x = getNoteXPosition(getNoteByKeyAndOctave(0, i))
		dc.DrawString(fmt.Sprintf("C%d", i+1), (x + keyW/6), h-10)
	}
}

func prepareScreen(dc *gg.Context) {
	dc.SetRGB(0.17, 0.17, 0.17)
	dc.DrawRectangle(0, 0, w, h)
	dc.Fill()
}

func createFramesKeyboard() {
	var totalFrames = fps * int(math.Round(musicTime))
	for i := 0; i < totalFrames; i++ {
		var framePressedKeys = map[int]PlayingNote{}
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

	var frStr = fmt.Sprintf("%05d", i+1)

	if DEBUG == true {
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			log.Fatal(err)
		}

		face := truetype.NewFace(font, &truetype.Options{Size: 9})

		dc.SetFontFace(face)
		dc.DrawString(fmt.Sprintf("FRAME %s", frStr), 30, 30)
	}

	dc.SavePNG(fmt.Sprintf("_frames/fr%s.png", frStr))
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
			if int(f)%(fps*30) == 0 {
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
	const maxWorkers = 100
	sem := make(chan struct{}, maxWorkers)

	files, err := filepath.Glob("_frames/fr*.png")
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		wg.Add(1)
		sem <- struct{}{}

		go func(f string) {
			if err := os.Remove(f); err != nil {
				log.Fatal(err)
			}
			<-sem
			wg.Done()
		}(f)
	}

	wg.Wait()
}

func prepareMidi(midiData midiparser.ParsedMidi) {
	var quarterNoteTicks = midiData.Meta.QuarterValue

	var skipChannels = map[byte]bool{}
	for channelId, channel := range midiData.Channels {
		if channel.Patch > 80 {
			skipChannels[channelId] = true
		}
	}

	for _, tempo := range midiData.Meta.Tempos {
		setTickBpm(tempo.OnTick, tempo.Bpm)

		var onTick = tempo.OnTick
		var onTickTime = getTickTime(onTick, quarterNoteTicks)
		var onTickFrame = math.Round(onTickTime * float64(fps))
		setFrameBpmChange(int(onTickFrame), tempo.Bpm)
	}

	for trackIndex, track := range midiData.Tracks {
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

			setNoteAction(note, int(onTickFrame), int(offTickFrame), trackIndex)
		}

		var trackTimeSeconds = getTickTime(track.Time, quarterNoteTicks)
		if trackTimeSeconds > musicTime {
			musicTime = trackTimeSeconds
		}
	}
}

func Generate() {
	executionStartTime := time.Now()
	f, err := os.Open(midiFilePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	parsedMidi, err := midiparser.ParseFile(f)
	if err != nil {
		panic(err)
	}

	outputMp3Path := convertMidiToMp3(midiFilePath)

	defer removeAudioFile(outputMp3Path)
	prepareMidi(parsedMidi)
	createFramesKeyboard()
	createFrames()
	defer removeFrames()

	err = createVideoFromFrames(framesFolderPath, outputMp3Path, outputVideoPath)
	if err != nil {
		log.Fatal(err)
	}

	executionTime := time.Since(executionStartTime)

	fmt.Printf("Execution time: %f seconds\nVideo Generated: %s\n", executionTime.Seconds(), outputVideoPath)

}
