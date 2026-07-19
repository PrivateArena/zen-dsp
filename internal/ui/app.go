package ui

import (
	"fmt"
	"strings"

	"zen-dsp/internal/config"
	"zen-dsp/internal/preset"
	"zen-dsp/internal/spa"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppState struct {
	Preset    preset.Preset
	BandIndex int
	Dirty     bool
	PWOK      bool
	EQNode    uint32
}

type App struct {
	tui   *tview.Application
	state AppState
}

func NewApp() *App {
	a := &App{tui: tview.NewApplication()}
	a.state = AppState{
		Preset:    flat(),
		BandIndex: 0,
		PWOK:      true,
	}
	return a
}

func flat() preset.Preset {
	p := preset.Preset{Name: "Flat", PreampDB: 0, SrcHz: 48000}
	p.Filters = []preset.Band{
		{Index: 0, Type: preset.BQPeaking, FreqHz: 32, GainDB: 0, Q: 1.0},
		{Index: 1, Type: preset.BQPeaking, FreqHz: 64, GainDB: 0, Q: 1.0},
		{Index: 2, Type: preset.BQPeaking, FreqHz: 125, GainDB: 0, Q: 1.0},
		{Index: 3, Type: preset.BQPeaking, FreqHz: 250, GainDB: 0, Q: 1.0},
		{Index: 4, Type: preset.BQPeaking, FreqHz: 500, GainDB: 0, Q: 1.0},
		{Index: 5, Type: preset.BQPeaking, FreqHz: 1000, GainDB: 0, Q: 1.0},
		{Index: 6, Type: preset.BQPeaking, FreqHz: 2000, GainDB: 0, Q: 1.0},
		{Index: 7, Type: preset.BQPeaking, FreqHz: 4000, GainDB: 0, Q: 1.0},
		{Index: 8, Type: preset.BQPeaking, FreqHz: 8000, GainDB: 0, Q: 1.0},
		{Index: 9, Type: preset.BQPeaking, FreqHz: 16000, GainDB: 0, Q: 1.0},
	}
	return p
}

func (a *App) Run() error {
	v := tview.NewTextView()
	v.SetDynamicColors(true)
	v.SetBorder(true)
	v.SetTitle(" zen-dsp ")
	v.SetWrap(false)
	v.SetChangedFunc(func() { a.tui.Draw() })

	v.SetText(RenderView(&a.state))

	a.tui.SetRoot(v, true)
	a.tui.SetInputCapture(a.capture(v))
	return a.tui.Run()
}

func (a *App) capture(v *tview.TextView) func(*tcell.EventKey) *tcell.EventKey {
	return func(ev *tcell.EventKey) *tcell.EventKey {
		if ev == nil {
			return nil
		}
		r := ev.Rune()
		k := ev.Key()
		switch {
		case r == 'e':
			a.adjust(+1, 0, 0)
		case r == 'd':
			a.adjust(-1, 0, 0)
		case r == 's':
			a.adjust(0, -1, 0)
		case r == 'f':
			a.adjust(0, +1, 0)
		case r == 'w':
			a.adjust(0, 0, -1)
		case r == 'r':
			a.adjust(0, 0, +1)
		case k == tcell.KeyTab:
			a.cycleType(+1)
		case k == tcell.KeyBacktab:
			a.cycleType(-1)
		case r == 'j':
			a.selectBand(+1)
		case r == 'k':
			a.selectBand(-1)
		case k == tcell.KeyDown:
			a.selectBand(+1)
		case k == tcell.KeyUp:
			a.selectBand(-1)
		case k == tcell.KeyRight:
			a.adjust(0, +1, 0)
		case k == tcell.KeyLeft:
			a.adjust(0, -1, 0)
		case r == ':':
			a.cmdMode(v)
		case r == 'q':
			a.tui.Stop()
		}
		a.refresh(v)
		return ev
	}
}

func (a *App) refresh(v *tview.TextView) {
	v.SetText(RenderView(&a.state))
}

func (a *App) adjust(gain, freq, q int) {
	b := a.selectedBand()
	if b == nil {
		return
	}
	b.GainDB = clampf(b.GainDB+float64(gain)*1, -24, 24)
	switch freq {
	case +1:
		b.FreqHz = nextFreq(b.FreqHz)
	case -1:
		b.FreqHz = prevFreq(b.FreqHz)
	}
	switch q {
	case +1:
		b.Q = clampf(b.Q+0.1, 0.1, 20)
	case -1:
		b.Q = clampf(b.Q-0.1, 0.1, 20)
	}
	a.state.Dirty = true
}

func nextFreq(v float64) float64 {
	factors := []float64{1.05, 1.1, 1.2, 1.25, 1.5, 2, 2.5, 3, 4, 5}
	for _, f := range factors {
		if v*f <= 20000 {
			return v * f
		}
	}
	return 20000
}

func prevFreq(v float64) float64 {
	factors := []float64{1.05, 1.1, 1.2, 1.25, 1.5, 2, 2.5, 3, 4, 5}
	for _, f := range factors {
		if v/f >= 20 {
			return v / f
		}
	}
	return 20
}

func (a *App) cycleType(dir int) {
	b := a.selectedBand()
	if b == nil {
		return
	}
	types := []preset.FilterType{
		preset.BQLowshelf, preset.BQHighshelf, preset.BQLowpass, preset.BQHighpass,
		preset.BQPeaking, preset.BQNotch, preset.BBandpass, preset.BQAllpass,
	}
	for i, t := range types {
		if t == b.Type {
			b.Type = types[(i+dir+len(types))%len(types)]
			break
		}
	}
	a.state.Dirty = true
}

func (a *App) selectBand(dir int) {
	n := len(a.state.Preset.Filters)
	if n == 0 {
		return
	}
	a.state.BandIndex = (a.state.BandIndex + dir + n) % n
}

func (a *App) selectedBand() *preset.Band {
	i := a.state.BandIndex
	if i < 0 || i >= len(a.state.Preset.Filters) {
		return nil
	}
	return &a.state.Preset.Filters[i]
}

func (a *App) cmdMode(v *tview.TextView) {
	input := tview.NewInputField()
	input.SetLabel(":").
		SetFieldWidth(30)
	f := tview.NewForm().
		AddInputField("cmd", "", 30, nil, func(t string) {
			switch t {
			case "w":
				a.save()
			case "x":
				a.apply()
			case "q":
				a.tui.Stop()
			}
			a.tui.SetRoot(v, true)
			a.refresh(v)
		}).
		AddButton("Cancel", func() {
			a.tui.SetRoot(v, true)
		})
	a.tui.SetRoot(f, true)
	a.tui.SetFocus(f)
}

func (a *App) save() error {
	b, err := a.state.Preset.ToJSON()
	if err != nil {
		return err
	}
	if err := config.WritePreset(b); err != nil {
		return err
	}
	a.state.Dirty = false
	return nil
}

func (a *App) apply() error {
	_, err := spa.FilterChainConf(a.state.Preset)
	if err != nil {
		return err
	}
	_ = config.AtomicWrite(config.XDGPipeWireConf(), "placeholder")
	a.state.Dirty = false
	return nil
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func RenderView(s *AppState) string {
	p := s.Preset
	var sb strings.Builder
	sb.WriteString(RenderGraphText(p, 60, 12))
	sb.WriteString("\n")
	sb.WriteString(renderBands(p, s.BandIndex))
	sb.WriteString("Mode: TUI   SR: ")
	sb.WriteString(fmt.Sprintf("%d", p.SrcHz))
	sb.WriteString("   Sink: zen-dsp.eq\n")
	sb.WriteString("[yellow]e/d[-] gain  [yellow]s/f[-] freq  [yellow]w/r[-] Q  [yellow]Tab[-] type  [yellow]j/k[-] select  [yellow]:w[-] save  [yellow]:x[-] apply  [yellow]:q[-] quit\n")
	sb.WriteString("[red]?[-] for help\n")
	return sb.String()
}

func renderBands(p preset.Preset, sel int) string {
	var sb strings.Builder
	for i, b := range p.Filters {
		mark := "  "
		if i == sel {
			mark = "> "
		}
		fmt.Fprintf(&sb, "%s[%d] %-10s %6gHz %+6.1fdB Q%.2g\n", mark, i, b.Type, b.FreqHz, b.GainDB, b.Q)
	}
	return sb.String()
}
