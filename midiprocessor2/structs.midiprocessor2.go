package midiprocessor2

type ParsedMidi struct {
	Header struct {
		KeySignatures []struct {
			Key   string `json:"key"`
			Scale string `json:"scale"`
			Ticks int    `json:"ticks"`
		} `json:"keySignatures"`
		Meta   []interface{} `json:"meta"`
		Name   string        `json:"name"`
		Ppq    int           `json:"ppq"`
		Tempos []struct {
			Bpm   float64 `json:"bpm"`
			Ticks int     `json:"ticks"`
		} `json:"tempos"`
		TimeSignatures []struct {
			Ticks         int   `json:"ticks"`
			TimeSignature []int `json:"timeSignature"`
			Measures      int   `json:"measures"`
		} `json:"timeSignatures"`
	} `json:"header"`
	Tracks []struct {
		Channel        int `json:"channel"`
		ControlChanges struct {
			Num7 []struct {
				Number int `json:"number"`
				Ticks  int `json:"ticks"`
				Time   int `json:"time"`
				Value  int `json:"value"`
			} `json:"7"`
			Num64 []struct {
				Number int `json:"number"`
				Ticks  int `json:"ticks"`
				Time   int `json:"time"`
				Value  int `json:"value"`
			} `json:"64"`
		} `json:"controlChanges"`
		PitchBends []interface{} `json:"pitchBends"`
		Instrument struct {
			Family string `json:"family"`
			Number int    `json:"number"`
			Name   string `json:"name"`
		} `json:"instrument"`
		Name  string `json:"name"`
		Notes []struct {
			Duration      float64 `json:"duration"`
			DurationTicks int     `json:"durationTicks"`
			Midi          int     `json:"midi"`
			Name          string  `json:"name"`
			Ticks         int     `json:"ticks"`
			Time          int     `json:"time"`
			Velocity      float64 `json:"velocity"`
		} `json:"notes"`
		EndOfTrackTicks int `json:"endOfTrackTicks"`
	} `json:"tracks"`
}
