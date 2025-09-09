package midiparser

type ParsedMidi struct {
	Tracks   []Track          `json:"tracks"`
	Channels map[byte]Channel `json:"channels"`
	Meta     HeaderMeta       `json:"meta"`
}
