package videogenerator

import (
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
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

func drawScreenAxes(dc *gg.Context) {
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

func createFrame(dc *gg.Context, i int) {
	var framePressedKeys = frameToPressedKeys[i]
	var frameFallingNotes = frameFallingNotes[i]
	prepareScreen(dc)
	drawScreenAxes(dc)
	drawKeyboard(dc, framePressedKeys)
	drawCNotesNotation(dc)
	drawFallingNotes(dc, frameFallingNotes)

	var frStr = fmt.Sprintf("%05d", i+1)
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
