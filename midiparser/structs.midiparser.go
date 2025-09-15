package midiparser

type Event struct {
	Note    int  `json:"note"`
	OnTick  int  `json:"on_tick"`
	Offtick int  `json:"off_tick"`
	Channel byte `json:"channel"`
	Meta    Meta `json:"meta"`
}

type Channel struct {
	Name       string `json:"name"`
	Instrument string `json:"instrument"`
	Patch      byte   `json:"patch"`
}

type Meta struct {
	Bpm float64 `json:"bpm"`
}

type Track struct {
	Events []Event
	Time   int
}

type Tempo struct {
	Bpm    float64 `json:"bpm"`
	OnTick int     `json:"on_tick"`
}
type HeaderMeta struct {
	QuarterValue int     `json:"quarterValue"`
	TracksNumber int     `json:"tracksNumber"`
	Tempos       []Tempo `json:"tempos"`
}

type ParsedMidi struct {
	Tracks   []Track          `json:"tracks"`
	Channels map[byte]Channel `json:"channels"`
	Meta     HeaderMeta       `json:"meta"`
}
