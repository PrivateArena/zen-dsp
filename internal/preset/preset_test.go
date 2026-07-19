package preset

import (
	"testing"
)

func TestDefaultFlatCount(t *testing.T) {
	p := DefaultFlat(48000)
	if len(p.Filters) != 10 {
		t.Fatalf("expected 10 bands, got %d", len(p.Filters))
	}
	if p.PreampDB != 0 {
		t.Fatalf("preamp should start at 0, got %g", p.PreampDB)
	}
}

func TestPresetRoundTrip(t *testing.T) {
	p := DefaultFlat(48000)
	p.Filters[0].GainDB = 3.5
	b, err := p.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	out, err := PresetFromJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != p.Name || len(out.Filters) != len(p.Filters) {
		t.Fatalf("round trip mismatch")
	}
	if out.Filters[0].GainDB != 3.5 {
		t.Fatalf("gainDB not preserved: %g", out.Filters[0].GainDB)
	}
}

func TestCloneIsolation(t *testing.T) {
	p := DefaultFlat(48000)
	c := p.Clone()
	c.Filters[0].GainDB = 99
	if p.Filters[0].GainDB != 0 {
		t.Fatalf("clone mutated original")
	}
}

func TestValidateRejectsHighQ(t *testing.T) {
	b := Band{Index: 0, Type: BQPeaking, FreqHz: 5000, GainDB: 0, Q: 50}
	errs := ValidateBand(b, 48000)
	found := false
	for _, e := range errs {
		if e == ErrUnstableBiquad || e == ErrQOutOfRange {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stability or Q range error, got %v", errs)
	}
}

func TestLogFreqMapping(t *testing.T) {
	x := LogFreqX(1000, 20, 20000, 100)
	if x == 0 || x > 100 {
		t.Fatalf("unexpected log freq mapping: %d", x)
	}
}
