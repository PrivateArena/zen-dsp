package spa

import (
	"strings"
	"testing"

	"zen-dsp/internal/preset"
)

func TestLavfiFormatting(t *testing.T) {
	p := preset.Preset{
		Name:     "Jazz",
		PreampDB: 3,
		SrcHz:    48000,
		Filters: []preset.Band{
			{Index: 0, Type: preset.BQLowshelf, FreqHz: 200, GainDB: 3, Q: 0.7},
			{Index: 1, Type: preset.BQPeaking, FreqHz: 2000, GainDB: -2, Q: 1.4},
		},
	}
	chain, err := FormatFilterChainString(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(chain))
	}
	if chain[0] != "lavfi.aveq.num_filters=2" {
		t.Fatalf("bad num_filters: %s", chain[0])
	}
	if chain[1] != "lavfi.aveq.preamp=3" {
		t.Fatalf("bad preamp: %s", chain[1])
	}
	if !strings.HasPrefix(chain[2], "lavfi.aveq.filter_0=") {
		t.Fatalf("bad filter prefix: %s", chain[2])
	}
}

func TestUnsupportedType(t *testing.T) {
	p := preset.Preset{Filters: []preset.Band{{Type: "bq_weird"}}}
	if _, err := FormatFilterChainString(p); err == nil {
		t.Fatalf("expected error for unsupported type")
	}
}
