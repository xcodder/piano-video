package midiparser

import (
	"fmt"

	// "io/ioutil"
	"log"
	"os"
	"strconv"
)

func readMoreBytes(bytes int) func(f *os.File) int {
	return func(f *os.File) int {
		readBytes(f, bytes)
		return bytes
	}
}

func bytesToString(bts []byte) string {
	str := ""
	for _, s := range bts {
		str += string(s)
	}
	return str
}

type Meta struct {
	Bpm float64 `json:"bpm"`
}

func readText() func(f *os.File) int {
	return func(f *os.File) int {
		len := readBytes(f, 1)[0]
		bytesToString(readBytes(f, int(len)))
		// allChannels
		// fmt.Println(name)
		return int(len) + 1
	}
}
func setBpm(f *os.File) int {
	readBytes(f, 1) // irrelevant byte
	bpm := float64(bytesToInt(readBytes(f, 3)))
	bpm = 60000000 / bpm
	// fmt.Println("BPM value: ", bpm, " On track: ", trackIndex)
	allTracks[trackIndex].Events = append(allTracks[trackIndex].Events, Event{OnTick: allTracks[trackIndex].Time, Meta: Meta{Bpm: bpm}})

	return 4
}
func timeSig(f *os.File) int {
	readBytes(f, 1) // irrelevant byte
	tSig := readBytes(f, 4)
	fmt.Println("Time Signature: ", tSig)
	return 5
}
func offset(f *os.File) int {
	readBytes(f, 1) // irrelevant byte
	offs := readBytes(f, 5)
	fmt.Println("Offset: ", offs)
	return 6
}
func midiChannelPrefix(f *os.File) int {
	readBytes(f, 1) // irrelevant byte
	readBytes(f, 1)
	return 2
}
func midiPort(f *os.File) int {
	readBytes(f, 1) // irrelevant byte
	readBytes(f, 1)
	return 2
}

var FFevents = map[byte]func(f *os.File) int{
	0:   readMoreBytes(1),
	1:   readText(),
	2:   readText(),
	3:   readText(),
	4:   readText(),
	5:   readText(),
	6:   readText(),
	7:   readText(),
	8:   readText(),
	9:   readText(),
	32:  midiChannelPrefix,
	33:  midiPort,
	47:  readMoreBytes(1),
	81:  setBpm,
	84:  offset,
	88:  timeSig,
	89:  readMoreBytes(3),
	127: readText(),
}

type Event struct {
	Note    int  `json:"note"`
	OnTick  int  `json:"on_tick"`
	Offtick int  `json:"off_tick"`
	Channel byte `json:"channel"`
	Meta    Meta `json:"meta"`
}

var allChannels = make(map[byte]Channel)

type Channel struct {
	Name       string `json:"name"`
	Instrument string `json:"instrument"`
	Patch      byte   `json:"patch"`
}

func statusf0(f *os.File, channel byte) int {
	bytesLen, bytesRead := readVarLen(f)
	readBytes(f, bytesLen)
	return bytesLen + bytesRead
}

var events = map[byte]func(f *os.File, channel byte) int{
	255: func(f *os.File, channel byte) int {
		n := readBytes(f, 1)[0]
		fn, ok := FFevents[n]
		if ok {
			return fn(f) + 1
		}
		log.Fatal("Unknown status value for FF ", n)
		return 0
	},
	240: statusf0,
	128: func(f *os.File, channel byte) int {
		key := readBytes(f, 2)[0]
		var index int
		for i := len(allTracks[trackIndex].Events) - 1; i > 0; i-- {
			n := allTracks[trackIndex].Events[i]
			if n.Note == int(key) {
				index = i
				break
			}
		}
		if trackIndex == 2 {
			fmt.Println(index, trackIndex, allTracks[trackIndex].Time)
		}
		allTracks[trackIndex].Events[index].Offtick = allTracks[trackIndex].Time
		c := 2
		return c
	},
	144: func(f *os.File, channel byte) int {
		bts := readBytes(f, 2)
		key := bts[0]
		velocity := bts[1]
		on := velocity != 0
		if on {
			allTracks[trackIndex].Events = append(allTracks[trackIndex].Events, Event{Note: int(key), OnTick: allTracks[trackIndex].Time, Channel: channel})
		} else {
			var index int
			for i := len(allTracks[trackIndex].Events) - 1; i > 0; i-- {
				n := allTracks[trackIndex].Events[i]
				if n.Note == int(key) {
					index = i
					break
				}
			}
			allTracks[trackIndex].Events[index].Offtick = allTracks[trackIndex].Time
		}
		c := 2
		return c
	},
	176: func(f *os.File, channel byte) int {
		readBytes(f, 2)
		c := 2
		return c
	},
	192: func(f *os.File, channel byte) int {
		patchNumber := readBytes(f, 1)[0]
		instrument := instrumentsTable[int(patchNumber)]
		if instrument != "" {
			allChannels[channel] = Channel{Instrument: instrument, Patch: patchNumber}
		}
		c := 1
		return c
	},
	224: func(f *os.File, channel byte) int {
		_ = readBytes(f, 2)[0]
		c := 2
		return c
	},
}
var buffer = make([]byte, 2020)

func readMThd(f *os.File) {
	readBytes(f, 8)
	trackFileFormat := bytesToInt(readBytes(f, 2))
	if trackFileFormat == 2 {
		log.Fatal("can't process track format 2")
	}
	tracksNumber := bytesToInt(readBytes(f, 2))
	headerMeta.TracksNumber = tracksNumber
	headerMeta.QuarterValue = bytesToInt(readBytes(f, 2))
}

var prependByte, lastStatus byte

func readBytes(f *os.File, bytes int) []byte {
	tmp := make([]byte, bytes)
	if prependByte != 0 {
		tmp = tmp[:bytes-1]
		f.Read(tmp)
		tmp = append([]byte{prependByte}, tmp...)
		prependByte = 0
		return tmp
	}
	f.Read(tmp)
	return tmp
}
func bytesToInt(bytes []byte) int {
	str := ""
	for _, b := range bytes {
		str += fmt.Sprintf("%08b", b)
	}
	deltaSum, _ := strconv.ParseUint(str, 2, 32)
	return int(deltaSum)
}

func readVarLen(f *os.File) (int, int) {
	var value int
	var c byte
	bytesRead := 1
	value = int(readBytes(f, 1)[0])

	if value&0x80 != 0 {
		value &= 0x7F

		for {
			c = readBytes(f, 1)[0]
			bytesRead++

			value = (int(value) << 7) + int(c&0x7F)

			if c&0x80 == 0 {
				break
			}
		}
	}

	return int(value), bytesRead
}

func readEvent(f *os.File) int {
	deltaSumInt, deltaBytes := readVarLen(f)
	allTracks[trackIndex].Time += deltaSumInt

	status := readBytes(f, 1)[0]

	if status < 128 {
		prependByte = status
		channel := lastStatus & 15 // last 4 bits

		v, ok := events[lastStatus]
		if ok {
			return v(f, channel) + deltaBytes
		}
		eventId := lastStatus & 240 // first 4 bits
		v, ok = events[eventId]
		if ok {
			return v(f, channel) + deltaBytes
		}
		log.Fatal("ERROR unknown status: ", lastStatus)
	}
	lastStatus = status
	channel := status & 15  // last 4 bits
	eventId := status & 240 // first 4 bits

	v, ok := events[eventId]
	if ok && status < 240 {
		return v(f, channel) + deltaBytes + 1
	}
	v, ok = events[status]
	if ok {
		return v(f, channel) + deltaBytes + 1
	}

	log.Fatal("ERROR STATUS: ", status)
	return 0
}
func readChunk(f *os.File) {
	buffer = buffer[:4]
	f.Read(buffer)
	// chunkId := string(buffer)
	buffer = buffer[:4]
	f.Read(buffer)
	chunkBytes := bytesToInt(buffer)
	bytesRead := 0
	for bytesRead < chunkBytes {
		bytesRead += readEvent(f)
	}
	if bytesRead != chunkBytes {
		log.Fatal("No match. Must be a bug or corrupt midi: Read should Be: ", bytesRead, "==", chunkBytes)
	}
}

var trackIndex = 0

type Track struct {
	Events []Event
	Time   int
}

var allTracks = []Track{}

type HeaderMeta struct {
	QuarterValue int `json:"quarterValue"`
	TracksNumber int `json:"tracksNumber"`
}

var headerMeta = HeaderMeta{}

func ParseFile(f *os.File) (ParsedMidi, error) {
	// f, err := os.Open(file)
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	readMThd(f)
	tracksNumber := headerMeta.TracksNumber
	allTracks = make([]Track, tracksNumber)
	for i := 0; i < tracksNumber; i++ {
		readChunk(f)
		trackIndex++
	}
	var parsedMidi = ParsedMidi{
		Tracks:   allTracks,
		Channels: allChannels,
		Meta:     headerMeta,
	}

	return parsedMidi, nil
	// rankingsJSON, _ := json.Marshal(allData)
	// os.WriteFile("./midioutput.json", rankingsJSON, 0644)
}
