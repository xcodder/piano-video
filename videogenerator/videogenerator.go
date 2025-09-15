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
	"time"
	"videos2/midiparser"
)

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
