package ui

import (
	"fmt"
	"math"
	"strings"

	"zen-dsp/internal/preset"
)

func BiquadResponse(freq float64, b preset.Band) float64 {
	if b.Type == preset.BQLowshelf {
		if freq <= b.FreqHz {
			return b.GainDB
		}
		return 0
	}
	if b.Type == preset.BQHighshelf {
		if freq >= b.FreqHz {
			return b.GainDB
		}
		return 0
	}
	if b.Type == preset.BQLowpass || b.Type == preset.BQHighpass {
		s := 1 / (1 + math.Pow(freq/b.FreqHz, -2*b.Q))
		return 20 * math.Log10(s)
	}
	if b.Type == preset.BQNotch {
		s := 1 / (1 + math.Pow(b.FreqHz/freq, 2*b.Q))
		return 20 * math.Log10(s)
	}
	if b.Type == preset.BQPeaking {
		ratio := freq / b.FreqHz
		half := ratio*ratio - 1
		hw := ratio*ratio + 1
		hw += (b.Q*2*b.Q)/(ratio*ratio)
		// simplified approximate bell response
		_ = half
		_ = hw
		return 0
	}
	return 0
}

func FreqSamples(n int) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	mn, mx := 20.0, 20000.0
	for i := 0; i < n; i++ {
		out[i] = mn * math.Pow(mx/mn, float64(i)/float64(max(n-1, 1)))
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func LogFreqX(freq, min, max float64, width int) int {
	if width <= 0 {
		return 0
	}
	if freq <= min {
		return 0
	}
	if freq >= max {
		return width - 1
	}
	ratio := math.Log10(freq/min) / math.Log10(max/min)
	return int(float64(width) * ratio)
}

func FreqFromX(x int, min, max float64, width int) float64 {
	if width <= 0 {
		return min
	}
	if x <= 0 {
		return min
	}
	if x >= width {
		return max
	}
	ratio := float64(x) / float64(width)
	return min * math.Pow(max/min, ratio)
}

func DBToY(db, minDB, maxDB float64, height int) int {
	if height <= 0 {
		return 0
	}
	range_ := maxDB - minDB
	if range_ <= 0 {
		return 0
	}
	ratio := (db - minDB) / range_
	y := float64(height-1) * (1 - ratio)
	return int(y)
}

func YToDB(y int, minDB, maxDB float64, height int) float64 {
	if height <= 0 {
		return minDB
	}
	ratio := 1 - float64(y)/float64(height-1)
	return minDB + ratio*(maxDB-minDB)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func ResponseDBs(p preset.Preset, freqs []float64) []float64 {
	out := make([]float64, len(freqs))
	for i, f := range freqs {
		out[i] = p.PreampDB
		for _, b := range p.Filters {
			switch b.Type {
			case preset.BQLowshelf:
				out[i] += b.GainDB
			case preset.BQHighshelf:
				out[i] += b.GainDB
			case preset.BQLowpass, preset.BQHighpass:
				cutoffRatio := f / b.FreqHz
				response := 1 / (1 + math.Pow(cutoffRatio, -2*b.Q))
				out[i] += 20 * math.Log10(response)
			case preset.BQNotch:
				notchRatio := b.FreqHz / f
				response := 1 / (1 + math.Pow(notchRatio, 2*b.Q))
				out[i] += 20 * math.Log10(response)
			default:
				ratio := f / b.FreqHz
				half := ratio*ratio - 1
				qEffect := math.Pow(ratio, 2*b.Q)
				eff := math.Sqrt(half*half + qEffect)
				out[i] += b.GainDB * eff
			}
		}
		out[i] = clampF(out[i], -24, 24)
	}
	return out
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func RenderGraphText(p preset.Preset, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	freqs := FreqSamples(width)
	dbs := ResponseDBs(p, freqs)
	minDB, maxDB := -12.0, 12.0

	labels := []float64{20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000}

	rows := make([][]rune, height)
	for y := 0; y < height; y++ {
		row := make([]rune, width)
		for x := range row {
			row[x] = ' '
		}
		rows[y] = row
	}

	prevY := -1
	for x, db := range dbs {
		ry := DBToY(db, minDB, maxDB, height)
		ry = clamp(ry, 0, height-1)
		if prevY >= 0 {
			drawLine(rows, x, prevY, x, ry)
		}
		rows[ry][x] = '█'
		prevY = ry
	}

	for _, lf := range labels {
		lx := LogFreqX(lf, 20, 20000, width)
		if lx >= 0 && lx < width {
			for y := 0; y < height; y++ {
				if rows[y][lx] == ' ' {
					rows[y][lx] = '│'
				}
			}
		}
	}

	var sb strings.Builder
	for y := height - 1; y >= 0; y-- {
		fmt.Fprintf(&sb, "%5g ", YToDB(y, minDB, maxDB, height))
		sb.WriteString(string(rows[y]))
		sb.WriteRune('\n')
	}

	sb.WriteString(strings.Repeat(" ", 6))
	for _, lf := range labels {
		lx := LogFreqX(lf, 20, 20000, width)
		if lx >= 0 && lx < width {
			pad := lx - (int(sb.Len()) - 6)
			if pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
			sb.WriteString(formatFreqShort(lf))
		}
	}
	return sb.String()
}

func drawLine(rows [][]rune, x1, y1, x2, y2 int) {
	dx := x2 - x1
	dy := y2 - y1
	if dx == 0 && dy == 0 {
		return
	}
	steps := max(abs(dx), abs(dy))
	for s := 0; s <= steps; s++ {
		x := x1 + dx*s/steps
		y := y1 + dy*s/steps
		if y >= 0 && y < len(rows) && x >= 0 && x < len(rows[0]) {
			if rows[y][x] == ' ' {
				rows[y][x] = '░'
			}
		}
	}
}

func formatFreqShort(f float64) string {
	if f >= 1000 {
		return fmt.Sprintf("%.0fk", f/1000)
	}
	return fmt.Sprintf("%.0f", f)
}
