package videogenerator

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
	Track  int
}

type PlayingNote struct {
	Active bool
	Track  int
}

type Color struct {
	R float64
	G float64
	B float64
}

type ScreenResolution [2]float64
