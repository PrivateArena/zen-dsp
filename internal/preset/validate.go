package preset

import (
	"errors"
	"math"
)

var (
	ErrFreqOutOfRange  = errors.New("frequency out of range")
	ErrGainOutOfRange  = errors.New("gain out of range")
	ErrQOutOfRange     = errors.New("Q out of range")
	ErrUnstableBiquad  = errors.New("Q exceeds stability limit near Nyquist")
	ErrIndexOutOfRange = errors.New("filter index out of range")
)

func ValidateBand(b Band, sr uint32) []error {
	var errs []error
	if b.FreqHz < 20 || b.FreqHz > float64(sr/2) {
		errs = append(errs, ErrFreqOutOfRange)
	}
	if b.GainDB < -24 || b.GainDB > 24 {
		errs = append(errs, ErrGainOutOfRange)
	}
	if b.Q < 0.1 || b.Q > 20 {
		errs = append(errs, ErrQOutOfRange)
	}
	if b.Index > 31 {
		errs = append(errs, ErrIndexOutOfRange)
	}
	if len(errs) == 0 && b.FreqHz > 0 {
		nyquist := float64(sr) / 2.0
		if nyquist > b.FreqHz {
			limit := b.FreqHz / (nyquist - b.FreqHz)
			if b.Q > limit && limit > 0 {
				errs = append(errs, ErrUnstableBiquad)
			}
		}
	}
	return errs
}

func FloorFreq(sr uint32) float64 { return 20 }
func CeilFreq(sr uint32) float64 { return float64(sr) / 2.0 }

func LogFreqX(freq, min, max float64, width int) int {
	if freq <= min {
		return 0
	}
	if freq >= max {
		return width
	}
	ratio := math.Log10(freq/min) / math.Log10(max/min)
	return int(math.Round(ratio * float64(width)))
}
