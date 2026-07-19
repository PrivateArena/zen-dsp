package main

import (
	"fmt"
	"os"

	"zen-dsp/internal/config"
	"zen-dsp/internal/preset"

	"zen-dsp/internal/ui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			if len(os.Args) < 3 || os.Args[2] != "init" {
				fmt.Fprintln(os.Stderr, "usage: zen-dsp config init")
				os.Exit(2)
			}
			p := preset.DefaultFlat(48000)
			b, _ := p.ToJSON()
			if err := config.WritePreset(b); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("initialized", config.XDGZenDSP())
			return
		case "headless":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: zen-dsp headless <preset.json>")
				os.Exit(2)
			}
			b, err := os.ReadFile(os.Args[2])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			p, err := preset.PresetFromJSON(b)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			// validate first
			for _, band := range p.Filters {
				if errs := preset.ValidateBand(band, p.SrcHz); len(errs) > 0 {
					for _, e := range errs {
						fmt.Fprintln(os.Stderr, "band", band.Index, e)
					}
					os.Exit(1)
				}
			}
			fmt.Println("ok")
			return
		}
	}

	// default: launch TUI
	app := ui.NewApp()
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
