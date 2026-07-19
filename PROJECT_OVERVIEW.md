<!-- codegraph-file-count: 14 -->
# zen-dsp — Go TUI Parametric EQ for PipeWire

## Purpose
A terminal-based parametric equaliser that generates PipeWire SPA-JSON filter chains. Users interactively sculpt EQ curves via keyboard-driven TUI (tview/tcell), persist presets as JSON to XDG config dirs, and apply them through PipeWire's `wpctl`/config drop-in mechanism. Written in Go 1.23.

## Architecture
```
User TUI (tview) → app.go event loop → preset.Band manipulation
    ↓ (save/apply)
config/writer.go  ──→  XDG config file (JSON preset)
spa/filter.go     ──→  PipeWire SPA-JSON filter chain string
```

## File Tree
```
zen-dsp/
├── main.go                    # CLI router — preset CRUD + DAW launch
├── cmd/
│   └── zen-dsp/main.go        # "daw" subcommand: TUI interactive EQ
├── internal/
│   ├── config/writer.go       # XDG-compliant atomic file I/O
│   ├── preset/
│   │   ├── preset.go          # Core domain model (Band, Preset, FilterType)
│   │   ├── validate.go        # Band bounds + log-freq mapping
│   │   └── preset_test.go     # Unit tests
│   ├── pw/check.go            # PipeWire runtime probe
│   ├── spa/
│   │   ├── filter.go          # SPA-JSON / lavfi.aveq string generation
│   │   └── filter_test.go     # Unit tests
│   └── ui/
│       ├── app.go             # TUI event loop, keyboard dispatch, rendering
│       ├── graph.go           # ASCII frequency-response graph
│       └── state.go           # Package declaration only
├── pkg/filterchain/chain_test.go  # SPA filter chain parse roundtrip
└── tests/integration/pw_test.go   # PipeWire smoke tests
```

## Component Roles

### Backend (data model + config + SPA generation)

| File / Module | Role | LOC | Key Exports (with signatures) |
|---|---|---|---|
| `internal/preset/preset.go` | Domain types: band, preset, filter type | ~82 | `FilterType string`; `Band struct {Freq,Gain,Q float64; Type FilterType}`; `Preset struct {SR uint32; Preamp float64; Bands []Band}`; `Clone() Preset`; `ToJSON() ([]byte, error)`; `PresetFromJSON(b []byte) (Preset, error)`; `DefaultFlat(sr uint32) Preset` |
| `internal/preset/validate.go` | Band validation + freq math | ~55 | `ValidateBand(b Band, sr uint32) []error`; `FloorFreq(sr uint32) float64`; `CeilFreq(sr uint32) float64`; `LogFreqX(freq, min, max float64, width int) int` |
| `internal/config/writer.go` | XDG atomic file read/write | ~71 | `AtomicWrite(path, content string) error`; `XDGPipeWireConf() string`; `XDGZenDSP() string`; `WritePreset(p []byte) error`; `LoadPreset() ([]byte, error)` |
| `internal/spa/filter.go` | SPA-JSON / lavfi chain generation | ~75 | `FormatFilterChainString(p preset.Preset) ([]string, error)`; `validType(t preset.FilterType) bool`; `SPAJSONProps(p preset.Preset) (map[string]interface{}, error)`; `FilterChainConf(p preset.Preset) (string, error)` |
| `internal/pw/check.go` | PipeWire availability check | ~12 | `Check() bool` |

### Frontend (TUI)

| File / Module | Role | LOC | Key Exports (with signatures) |
|---|---|---|---|
| `internal/ui/app.go` | TUI app: event loop, keyboard handlers, render | ~276 | `AppState struct`; `App struct`; `NewApp() *App`; `(a *App) Run() error`; `(a *App) capture(v *tview.TextView) func(*tcell.EventKey) *tcell.EventKey`; `(a *App) adjust(gain, freq, q int)`; `(a *App) cycleType(dir int)`; `(a *App) selectBand(dir int)`; `(a *App) save() error`; `(a *App) apply() error`; `RenderView(s *AppState) string`; `renderBands(p preset.Preset, sel int) string` |
| `internal/ui/graph.go` | ASCII frequency-response graph | ~257 | `BiquadResponse(freq float64, b preset.Band) float64`; `FreqSamples(n int) []float64`; `LogFreqX(freq, min, max float64, width int) int`; `FreqFromX(x int, min, max, width) float64`; `DBToY(db, minDB, maxDB, height) int`; `YToDB(y int, minDB, maxDB, height) float64`; `ResponseDBs(p preset.Preset, freqs []float64) []float64`; `RenderGraphText(p preset.Preset, width, height int) string` |
| `internal/ui/state.go` | Package declaration placeholder | ~1 | no exports |

### Entry points

| File / Module | Role | LOC | Key Exports (with signatures) |
|---|---|---|---|
| `main.go` | Root CLI: preset commands + DAW dispatch | ~72 | `main()`; `handleConfig(args []string)`; `loadPresetFile(path string) (preset.Preset, error)`; `applyString(p preset.Preset) (string, error)` |
| `cmd/zen-dsp/main.go` | `daw` subcommand: launch TUI | ~65 | `main()` |

## Cross-References

| File | Called by / calls | Why it's central |
|---|---|---|
| `internal/ui/app.go` | Calls: `preset.go` (DefaultFlat, ToJSON), `writer.go` (WritePreset), `spa/filter.go` (FilterChainConf), `graph.go` (RenderGraphText) | Central orchestrator: event loop dispatches all user actions to domain+config+SPA layers |
| `internal/ui/graph.go` | Called by: `app.go` (RenderView) | Core rendering: ASCII frequency-response graph consumes presets |
| `internal/preset/preset.go` | Called by: `app.go`, `cmd/zen-dsp/main.go`, `main.go`, `spa/filter.go`, tests | Single source of truth for the EQ data model |
| `internal/config/writer.go` | Called by: `main.go` (handleConfig), `app.go` (save, apply) | All persistent I/O (read/write preset, atomic file ops) funnels through this file |
| `main.go` | Calls: `writer.go`, `preset.go`, `spa/filter.go`, `app.go` (via subcommand) | CLI entry point: routes top-level args to subcommands and library packages |

## Key Architectural Patterns

1. **Preset as central transfer type**: Every layer (TUI, config persistence, SPA generation, validation) operates on the same `preset.Preset` struct — no DTO conversion between boundaries.
2. **Atomic file I/O via temp+rename**: `AtomicWrite` writes to a temp file in the same directory then renames, preventing partial writes from crashing PipeWire config drops.
3. **Keyboard-driven TUI modal loop**: `app.go` implements a single `capture()` event handler that dispatches to `adjust`/`cycleType`/`selectBand`/`cmdMode` based on key type — no external UI framework beyond tview's raw event model.
4. **SPA-JSON filter chain generation**: `spa/filter.go` converts the in-memory `Preset` to PipeWire's SPA JSON envelope string (`FilterChainConf`), composing both preamp and per-band biquad filters into a single SPA-JSON snippet.
5. **Dual entry-point architecture**: `main.go` handles CLI-only operations (show config, list, apply preset files) while `cmd/zen-dsp/main.go` launches the interactive TUI — both share the same backend packages.
6. **Log-frequency domain throughout**: All freq-related functions (`LogFreqX`, `FreqFromX`, `nextFreq`, `prevFreq`, `BiquadResponse`) operate in log space, matching human hearing perception and standard EQ UI conventions.

## Read Triggers

| If you need to... | Open these files |
|---|---|
| Add a new CLI subcommand | `main.go` (main/handleConfig), `cmd/zen-dsp/main.go` |
| Modify the EQ data model (new band field) | `internal/preset/preset.go`, `internal/preset/validate.go` |
| Change save/load preset location or format | `internal/config/writer.go` |
| Add a new filter type | `internal/preset/preset.go` (FilterType const), `internal/spa/filter.go` (validType), `internal/spa/filter_test.go` |
| Tweak the ASCII graph rendering | `internal/ui/graph.go` (RenderGraphText, BiquadResponse) |
| Add a new keyboard shortcut | `internal/ui/app.go` (capture method) |
| Modify PipeWire config output format | `internal/spa/filter.go` (FilterChainConf, FormatFilterChainString) |
| Change frequency range or stepping | `internal/preset/validate.go` (FloorFreq, CeilFreq), `internal/ui/app.go` (nextFreq, prevFreq) |
| Add integration test for PipeWire | `tests/integration/pw_test.go` |
| Check PipeWire runtime detection | `internal/pw/check.go` |

## Dependencies

### Runtime (Go)
| Package / Module | Role | Version |
|---|---|---|
| `github.com/rivo/tview` | TUI framework (flexbox, text views, input) | v0.42.0 |
| `github.com/gdamore/tcell/v2` | Terminal cell rendering backend | v2.8.1 |

### Transitive (indirect)
| Package / Module | Role |
|---|---|
| `golang.org/x/term`, `golang.org/x/sys` | Terminal raw mode, OS syscalls |
| `golang.org/x/text` | Unicode text transform |
| `github.com/mattn/go-runewidth`, `github.com/rivo/uniseg` | Character width / grapheme clusters |
| `github.com/gdamore/encoding`, `github.com/lucasb-eyer/go-colorful` | Terminal encoding, color conversion |

## Build & Run
| Command | Purpose |
|---|---|
| `go build .` | Build root CLI tool |
| `go build ./cmd/zen-dsp` | Build DAW subcommand binary |
| `go test ./...` | Run all unit + integration tests |
| `zen-dsp daw` | Launch interactive TUI parametric EQ |
