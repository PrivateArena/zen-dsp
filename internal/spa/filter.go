package spa

import (
	"fmt"
	"strings"

	"zen-dsp/internal/preset"
)

func FormatFilterChainString(p preset.Preset) ([]string, error) {
	out := []string{
		fmt.Sprintf("lavfi.aveq.num_filters=%d", len(p.Filters)),
		fmt.Sprintf("lavfi.aveq.preamp=%g", p.PreampDB),
	}
	for _, b := range p.Filters {
		if !validType(b.Type) {
			return nil, fmt.Errorf("unsupported filter type: %s", b.Type)
		}
		out = append(out, fmt.Sprintf("lavfi.aveq.filter_%d=%s:%g:%g:%g",
			b.Index, b.Type, b.FreqHz, b.Q, b.GainDB))
	}
	return out, nil
}

func validType(t preset.FilterType) bool {
	switch t {
	case preset.BQLowshelf, preset.BQHighshelf, preset.BQLowpass, preset.BQHighpass,
		preset.BQPeaking, preset.BQNotch, preset.BBandpass, preset.BQAllpass:
		return true
	}
	return false
}

func SPAJSONProps(p preset.Preset) (map[string]interface{}, error) {
	chain, err := FormatFilterChainString(p)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"type":         "api.alsa",
		"factory":      "filter-chain",
		"stream.props": map[string]interface{}{"filter.chain": chain},
	}, nil
}

func FilterChainConf(p preset.Preset) (string, error) {
	chain, err := FormatFilterChainString(p)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString("context.modules = [\n")
	sb.WriteString("  { name = libpipewire-module-filter-chain\n")
	sb.WriteString("    args = {\n")
	sb.WriteString("      capture.props = {\n")
	sb.WriteString("        node.name = \"zen-dsp.eq-capture\"\n")
	sb.WriteString("      }\n")
	sb.WriteString("      playback.props = {\n")
	sb.WriteString("        node.name = \"zen-dsp.eq\"\n")
	sb.WriteString("        media.class = \"Audio/Sink\"\n")
	sb.WriteString("      }\n")
	sb.WriteString("      filter.chain = [")
	for i, s := range chain {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("\"" + s + "\"")
	}
	sb.WriteString("]\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")
	return sb.String(), nil
}
