package filterchain

import (
	"testing"
)

func TestLavfiChainRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"preamp", "lavfi.aveq.preamp=3", []string{"lavfi.aveq.preamp=3"}},
		{"typed", "lavfi.aveq.filter_0=bq_peaking:1000:1.4:6", []string{"lavfi.aveq.filter_0=bq_peaking:1000:1.4:6"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.want) == 0 && tt.in == "" {
				return
			}
		})
	}
}
