package preset

import (
	"encoding/json"
)

type FilterType string

const (
	BQLowshelf  FilterType = "bq_lowshelf"
	BQHighshelf FilterType = "bq_highshelf"
	BQLowpass   FilterType = "bq_lowpass"
	BQHighpass  FilterType = "bq_highpass"
	BQPeaking   FilterType = "bq_peaking"
	BQNotch     FilterType = "bq_notch"
	BBandpass   FilterType = "bq_bandpass"
	BQAllpass   FilterType = "bq_allpass"
)

var AllFilterTypes = []FilterType{
	BQLowshelf, BQHighshelf, BQLowpass, BQHighpass,
	BQPeaking, BQNotch, BBandpass, BQAllpass,
}

type Band struct {
	Index  uint8      `json:"index"`
	Type   FilterType `json:"type"`
	FreqHz float64    `json:"freq_hz"`
	GainDB float64    `json:"gain_db"`
	Q      float64    `json:"q"`
}

type Preset struct {
	Name     string  `json:"name"`
	Filters  []Band  `json:"filters"`
	PreampDB float64 `json:"preamp_db"`
	SrcHz    uint32  `json:"sr_hz,omitempty"`
	Source   string  `json:"source,omitempty"`
}

func (p *Preset) Clone() Preset {
	out := *p
	if p.Filters != nil {
		out.Filters = make([]Band, len(p.Filters))
		copy(out.Filters, p.Filters)
	}
	return out
}

func (p *Preset) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func PresetFromJSON(b []byte) (Preset, error) {
	var p Preset
	if err := json.Unmarshal(b, &p); err != nil {
		return Preset{}, err
	}
	return p, nil
}

func DefaultFlat(sr uint32) Preset {
	return Preset{
		Name:     "Flat",
		PreampDB: 0,
		SrcHz:    sr,
		Source:   "user",
		Filters: []Band{
			{Index: 0, Type: BQPeaking, FreqHz: 32, GainDB: 0, Q: 1.0},
			{Index: 1, Type: BQPeaking, FreqHz: 64, GainDB: 0, Q: 1.0},
			{Index: 2, Type: BQPeaking, FreqHz: 125, GainDB: 0, Q: 1.0},
			{Index: 3, Type: BQPeaking, FreqHz: 250, GainDB: 0, Q: 1.0},
			{Index: 4, Type: BQPeaking, FreqHz: 500, GainDB: 0, Q: 1.0},
			{Index: 5, Type: BQPeaking, FreqHz: 1000, GainDB: 0, Q: 1.0},
			{Index: 6, Type: BQPeaking, FreqHz: 2000, GainDB: 0, Q: 1.0},
			{Index: 7, Type: BQPeaking, FreqHz: 4000, GainDB: 0, Q: 1.0},
			{Index: 8, Type: BQPeaking, FreqHz: 8000, GainDB: 0, Q: 1.0},
			{Index: 9, Type: BQPeaking, FreqHz: 16000, GainDB: 0, Q: 1.0},
		},
	}
}
