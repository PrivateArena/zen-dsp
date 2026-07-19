package main

import (
	"fmt"
	"os"

	"zen-dsp/internal/config"
	"zen-dsp/internal/preset"
	"zen-dsp/internal/spa"
)

func handleConfig(args []string) {
	if len(args) == 0 {
		fmt.Println(config.XDGPipeWireConf())
		return
	}
	switch args[0] {
	case "init":
		p := preset.DefaultFlat(48000)
		b, _ := p.ToJSON()
		if err := config.WritePreset(b); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("initialized", config.XDGZenDSP())
	default:
		fmt.Fprintln(os.Stderr, "usage: zen-dsp config init")
		os.Exit(2)
	}
}

func loadPresetFile(path string) (preset.Preset, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return preset.Preset{}, err
	}
	return preset.PresetFromJSON(b)
}

func applyString(p preset.Preset) (string, error) {
	return spa.FilterChainConf(p)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "config" {
		handleConfig(os.Args[2:])
		return
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "headless":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: zen-dsp headless <preset.json>")
				os.Exit(2)
			}
			p, err := loadPresetFile(os.Args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			s, err := applyString(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(s)
			return
		}
	}
	fmt.Println("zen-dsp: tui not implemented in this sketch")
}
