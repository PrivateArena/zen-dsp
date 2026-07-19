# zen-dsp v1 Architecture Plan

> Scope: Parametric EQ / DSP controller for Linux using pure PipeWire, no heavy Flatpak.  
> Stack: Go + tview. Integration via subprocess (pw-dump, pw-cli, wpctl) only.

---

## 1. System Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│                    zen-dsp (Go + tview)                    │
│                                                             │
│  ┌──────────────┐    ┌─────────────────┐                   │
│  │ Config Store │    │  Filter Engine  │                   │
│  │ (.conf/.apo) │    │  (filter chain) │                   │
│  └──────┬───────┘    └───────┬─────────┘                   │
│         │                   │                               │
│  ┌──────▼───────────────────▼─────────┐                    │
│  │         Transport Layer            │                    │
│  │  pw-dump (read) / pw-cli (write)   │                    │
│  └──────────────────┬────────────────┘                    │
│                     │                                      │
└─────────────────────┼──────────────────────────────────────┘
                      │
          ┌───────────▼────────────┐
          │     PipeWire Daemon    │
          │  libpipewire-module-   │
          │   filter-chain.so      │
          └───────────────────────┘
```

**Hard boundary**: zen-dsp does NOT process audio. It generates configuration that PipeWire's `libpipewire-module-filter-chain` applies.

**Why**: Avoid real-time thread safety, lock-free queues, and synchronization overhead. PipeWire already solves these.

---

## 2. Data Model

```pseudocode
struct Preset {
    name: string
    filters[]: Band
    preamp_db: float
    source: "user" | "autoeq"
}

struct Band {
    id: uint32
    type: FilterType   // peaking, low-shelf, high-shelf, low-pass, high-pass, notch, band-pass
    freq_hz: float     // 20 .. 20000
    gain_db: float     // -24 .. +24
    Q: float           // 0.1 .. 20.0
}

enum FilterType {
    PEAKING
    LOW_SHELF
    HIGH_SHELF
    LOW_PASS
    HIGH_PASS
    NOTCH
    BAND_PASS
}
```

### SPA-JSON wire format

```pseudocode
spa_json_format(preset) -> object {
    "type": "api.alsa",
    "factory": "filter-chain",
    "stream.props": {
        "filter.chain": filters.map(lambda f: format_filter(f))
    }
}

format_filter(band) -> object {
    "type": "eq.fir"  // or "eq.biquad" depending on complexity
    "band": band.type
    "freq": band.freq_hz
    "Q": band.Q
    "gain": band.gain_db
}
```

**Alternative considered**: Implement a custom PipeWire module in C.  
**Rejected**: Cross-compiling, packaging, and maintaining a shared object across distros is a maintenance burden. Process-level config is simpler.

---

## 3. TUI Layout (tview)

```
┌─── zen-dsp ─────────────────────────────────────┐
│ Filters [1]  [2]  [3]  [4]  [5]  [6]  [7]      │
│ ┌────────────────────────────────────────────┐   │
│ │ EQ Graph (tview-based canvas)              │   │
│ │ 24 ┤         ╱╲                            │   │
│ │ 12 ┤───────╱──╲──────╲                    │   │
│ │  0 ┤──────────────────────╲──╱───╱──╱────   │   │
│ │-12 ┤                                        │   │
│ │    20  50  100 200 500 1k 2k 5k 10k 20k     │   │
│ └────────────────────────────────────────────┘   │
│ [1] peaking  1000Hz  +6dB  Q=1.4               │
│ [2] low-shelf  200Hz  +3dB                      │
│ Controls:                                        │
│ <j/k> select band  <e/d> gain  <s/f> freq       │
│ <w/r> Q factor   <Tab> cycle type   <:> save    │
│                                                  │
│ Presets: [Flat] [Rock] [Jazz] [+ Add+]          │
│ Mode: tui  │  Output: ~/.config/pipewire/        │
└──────────────────────────────────────────────────┘
```

**Key binding philosophy**:  
- `esdf` for the currently selected band (shifted WASD).  
- `j/k` for vertical band selection, less strain on hand position.
- `:w` for vim-style save.
- Tab/Shift-Tab for type cycling.

### Frequency Scale Issue on Log10

```pseudocode
x_pixel = log10(freq_hz / MIN_HZ) / log10(MAX_HZ / MIN_HZ) * WIDTH_PX
freq_hz_from_pixel(x_pixel) = MIN_HZ * pow(10, (x_pixel / WIDTH_PX) * log10(MAX_HZ / MIN_HZ))
```

**Uncertainty**: The exact filter-chain JSON properties for per-band filter type. Candidate sources: `spa-json` parser from andyyu's repo, or WirePlumber docs.

---

## 4. Execution Flows

### 4.1 Save / Apply Configuration

```pseudocode
func ApplyPreset(p: Preset) -> error {
    conf := GenerateSPAJSON(p)
    path := DetermineOutputPath(p.name)  // XDG default or user-specified

    // Atomic write
    atomic_copy(conf, path)

    // Reload filter chain if PipeWire session is active
    // Option A: ask user to restart pipewire
    // Option B: pw-cli unload/load module chain (risky, may pop)
    // Chosen: Option A (reliability > live preview for v1)

    return nil
}
```

**Alternative considered**: Live-update via `pw-cli` module reload.  
**Rejected for v1**: Can cause audible pops. Safe approach for v1 is write-to-file + user restart.

### 4.2 Import Presets

```pseudocode
func Import(path string) -> (Preset, error) {
    ext := file_extension(path)

    if ext in [".apo", ".txt"] {
        return parse_apo_format(path)
    }
    elif ext in [".conf"] {
        return parse_spa_json(path)
    } else {
        return error(UnsupportedFormat)
    }
}
```

### 4.3 PipeWire Discovery

```pseudocode
func DiscoverActiveNodes() -> []Node {
    // pw-dump is json, parse stdout
    raw := exec_subprocess("pw-dump")
    nodes := parse_pw_dump(raw)

    // Filter for PLAYBACK targets with filter-chain factory available
    return nodes.filter(n -> n.can_chain_filter)
}
```

### 4.4 AutoEQ Import

```pseudocode
func FetchAutoEQ(target: string) -> error {
    // andyyu's repo uses an AutoEQ-equivalent API.
    // We can fetch from https://github.com/jaakkopasanen/AutoEq
    // and parse the CSV/RDS format.
    csv := http_get("https://raw.githubusercontent.com/..." + target + ".csv")
    preset := parse_autoeq_csv(csv)
    return preset
}
```

---

## 5. State Management

- **In-memory canonical state**: single `Preset` struct.  
- **Config file is derived write**: generated from in-memory state on save.  
- **Undo**: ring buffer of last N Preset snapshots.  
- **Dirty flag**: set on mutation; (:w!) force-saves; (:q without dirty → warn).

---

## 6. Process Communication Contract

| Process | Direction | Why subprocess, not CGO |
|---------|-----------|------------------------|
| `pw-dump` | Read | Pure JSON, no state mutation. Zero-copy pipe. |
| `pw-cli` | Write | Module load/unload via lazy evaluation if implemented. |
| `wpctl` | Control | Volume routing, node selection. |

**Rationale against CGO/PipeWire C bindings**:  
- Keeps binary static-linked, distro-agnostic.  
- Subprocess parsing is aggressively simpler than maintaining a GObject introspection layer in Go.

**Alternative considered**: go-pipewire (Go bindings).  
**Rejected**: Bindings lag system headers; smaller user base; harder to debug.

---

## 7. Configuration File Layout

```
~/.config/pipewire/pipewire.conf.d/
├── 99-zen-dsp-preset-flat.conf
└── 99-zen-dsp-preset-jazz.conf

~/.config/zen-dsp/
├── presets.json        # user-defined presets
├── last-session.conf    # tui state for reopen
└── keymap.toml         # optional keybinding overrides
```

**Steering file naming**: PipeWire applies `*.conf` in lexicographic order. Prefix `99-` ensures this loads after defaults.

---

## 8. Failure Modes

### 8.1 PipeWire not running

```
pw-dump: failed to connect to PipeWire
→ Show: "PipeWire daemon not detected. Is it running?"
→ Exit code: 2
→ No config written
```

### 8.2 Invalid filter-chain factory on target node

```
pw-cli error 98: No such interface/module
→ Show: "Node does not support filter-chain."
→ Fallback: offer to write .conf manually for user.
```

### 8.3 Write failure to XDG dir

```
os.Create: permission denied /home/user/.config/pipewire/...
→ Show: "Cannot write to PipeWire config dir. Check permissions."
→ Save fallback: write to /tmp/zen-dsp-backup.conf
```

### 8.4 Config syntax error in generated SPA-JSON

```
PipeWire restart rejects snippet.
→ Detect: pw-cli status shows error after restart.
→ Show: "Generated config rejected. Rolling back to backup."
→ Action: restore .bak copy
```

---

## 9. Concurrency Model

```
Main Thread: tview event loop
    │
    ├──► UI Rendering (sync, single goroutine)
    │
    └──► Background Worker goroutine
            │
            ├── pw-dump polling (optional, every N seconds)
            ├── preset save (async write)
            └── AutoEQ fetch (http.Client with timeout)
```

**Why single UI thread**: tview is not thread-safe. All writes to the app must come from the main goroutine via `app.QueueUpdate`.

---

## 10. Build & Distribution

```pseudocode
// Makefile targets
build:
    go build -o zen-dsp ./cmd/zen-dsp/

install:
    install -Dm755 zen-dsp /usr/local/bin/zen-dsp
    install -Dm644 pkg/desktop/zen-dsp.desktop /usr/share/applications/
```

**Distribution**: Native ELF binary. No container, no runtime dependency beyond libc and PipeWire (which is already on any PipeWire system).

**Alternative considered**: Flatpak.  
**Rejected**: Exactly the deployment model the requirement wants to avoid.

---

## 11. Open Questions

1. **Per-band biquad count**: Does `libpipewire-module-filter-chain` accept one biquad per band, or only FIR for full curves? Need to verify against PipeWire 1.0+ changelog.  
   - Confidence: 70% — requires docs check.

2. **Live reload without restart**: Is there an IPC-safe way to hot-swap filter-chain parameters?  
   - If yes, we can push params via `pw-cli` and skip restart.  
   - If no, file-based is the only path.

3. **Preset sharing format**: Support both `.apo` (RfishQ format) and native `.conf` for ecosystem compatibility.

4. **Headless / non-TUI mode**: For scripted config generation (useful for dotfiles).

---

## 12. Testing Strategy

```pseudocode
// Unit tests (no PipeWire required)
TestParseSPAJSON
TestFormatBand
TestLogFrequencyMapping
TestGenerateSPAJSON
TestParseAutoEQCSV

// Integration (requires PipeWire runtime)
TestApplyAndReadback
TestDiscoverActiveNodes

// Fuzz / property-based
TestPresetRoundTrip(gen_presets)
```

**Alternative considered**: Mock pw-dump.  
**Rejected for integration tests**: Real end-to-end with PIPE provided by PipeWire in CI container or local dev. Unit tests mock.

---

## 13. Pseudocode: Main Entry

```pseudocode
func main() {
    ctx := app.NewApplication()

    preset := &Preset{
        name: "Flat",
        filters: default_flat_bands,
        preamp_db: 0.0,
    }

    state := AppState{
        preset: preset,
        dirty: false,
        pipewire_detected: check_pipewire(),
    }

    if len(os.Args) > 1 && os.Args[1] == "config" {
        // config init / show path
        handle_config_subcommand()
        return
    }

    if len(os.Args) > 1 && os.Args[1] == "headless" {
        apply_and_exit(os.Args[2])
        return
    }

    // Interactive TUI
    draw_ui := func() {
        render_eq_graph(state)
        render_preset_list(state)
        render_current_band(state)
        render_keybindings(state)
    }

    ctx.SetInputCapture(handle_keypress_state_machine(state))

    if err := ctx.SetRoot(draw_ui).Run(); err != nil {
        log.Fatal(err)
    }

    if state.dirty {
        if !prompt_save_on_exit(ctx) {
            return
        }
        ApplyPreset(state.preset)
    }
}
```

---

## 14. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| PipeWire REST interface changes | Medium | High | Lock to pw-dump schema; pin compatible versions |
| tview unmaintained | Low | Medium | Pin version; accept port cost |
| Subprocess parsing drift | Medium | Medium | Snapshot exact pw-dump output for parser tests |
| SPA-JSON generation invalid | Medium | High | Round-trip test: generate → save → pw-cli parse → check error |

---

## 15. File Layout

```
zen-dsp/
├── cmd/
│   └── zen-dsp/
│       └── main.go
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   ├── parser.go
│   │   ├── writer.go
│   │   └── path.go
│   ├── preset/
│   │   ├── preset.go
│   │   ├── band.go
│   │   └── autoeq.go
│   ├── pw/
│   │   ├── dump.go
│   │   ├── cli.go
│   │   └── nodes.go
│   ├── ui/
│   │   ├── app.go
│   │   ├── eqgraph.go
│   │   ├── keymap.go
│   │   └── render.go
│   └── spa/
│       ├── format.go
│       └── filter.go
├── pkg/
│   ├── filterchain/
│   │   └── chain.go
│   └── desktop/
│       └── zen-dsp.desktop
├── examples/
│   └── presets/
│       ├── flat.conf
│       └── jazz.conf
├── go.mod
├── go.sum
└── README.md
```
