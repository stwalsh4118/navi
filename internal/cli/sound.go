package cli

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/stwalsh4118/navi/internal/audio"
)

const (
	soundTestDelay = 1500 * time.Millisecond
	exitOK         = 0
	exitError      = 1
)

var validEvents = []string{"waiting", "permission", "working", "idle", "stopped", "done", "error"}

// RunSound handles the `navi sound` subcommand.
func RunSound(args []string) int {
	if len(args) == 0 {
		printSoundUsage()
		return exitError
	}

	switch args[0] {
	case "test":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: navi sound test <event>")
			fmt.Fprintf(os.Stderr, "Valid events: %v\n", validEvents)
			return exitError
		}
		return runSoundTest(args[1])
	case "test-all":
		return runSoundTestAll()
	case "list":
		return runSoundList()
	default:
		fmt.Fprintf(os.Stderr, "Unknown sound subcommand: %q\n", args[0])
		printSoundUsage()
		return exitError
	}
}

func runSoundTest(event string) int {
	if !isValidEvent(event) {
		fmt.Fprintf(os.Stderr, "Unknown event: %q\nValid events: %v\n", event, validEvents)
		return exitError
	}

	cfg, err := audio.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return exitError
	}

	filePath, volume := resolveEventSound(cfg, event)
	if filePath == "" {
		fmt.Fprintf(os.Stderr, "No sound configured for event %q\n", event)
		return exitError
	}

	fmt.Printf("Event:  %s\nFile:   %s\nVolume: %d\n", event, filePath, volume)

	player := audio.NewPlayer(cfg.Player)
	if !player.Available() {
		fmt.Fprintln(os.Stderr, "No audio player available")
		return exitError
	}

	if err := player.Play(filePath, volume); err != nil {
		fmt.Fprintf(os.Stderr, "Error playing sound: %v\n", err)
		return exitError
	}

	// Wait for async playback to complete
	time.Sleep(soundTestDelay)
	return exitOK
}

func runSoundTestAll() int {
	cfg, err := audio.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return exitError
	}

	player := audio.NewPlayer(cfg.Player)
	if !player.Available() {
		fmt.Fprintln(os.Stderr, "No audio player available")
		return exitError
	}

	played := 0
	for _, event := range validEvents {
		if enabled, ok := cfg.Triggers[event]; !ok || !enabled {
			continue
		}

		filePath, volume := resolveEventSound(cfg, event)
		if filePath == "" {
			continue
		}

		fmt.Printf("%-12s %s (vol: %d)\n", event, filePath, volume)
		if err := player.Play(filePath, volume); err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			continue
		}
		played++
		time.Sleep(soundTestDelay)
	}

	if played == 0 {
		fmt.Println("No sounds configured for any enabled trigger")
	}
	return exitOK
}

func runSoundList() int {
	cfg, err := audio.LoadConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return exitError
	}

	packs, err := audio.ListPacks()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing packs: %v\n", err)
		return exitError
	}

	if len(packs) == 0 {
		fmt.Println("No sound packs found")
		fmt.Printf("Install packs to: %s\n", "~/.config/navi/soundpacks/")
		return exitOK
	}

	fmt.Println("Available sound packs:")
	for _, p := range packs {
		active := " "
		if p.Name == cfg.Pack {
			active = "*"
		}
		fmt.Printf("  %s %-20s %d events, %d files\n", active, p.Name, p.EventCount, p.FileCount)
	}

	if cfg.Pack != "" {
		fmt.Printf("\nActive pack: %s\n", cfg.Pack)
	} else {
		fmt.Println("\nNo active pack (using per-file config)")
	}

	return exitOK
}

func resolveEventSound(cfg *audio.Config, event string) (string, int) {
	volume := cfg.Volume.EffectiveVolume(event)

	// Files override takes precedence
	if filePath, ok := cfg.Files[event]; ok && filePath != "" {
		return filePath, volume
	}

	// Try pack
	if cfg.Pack != "" {
		files, err := audio.ResolveSoundFiles(cfg.Pack)
		if err == nil {
			if paths, ok := files[event]; ok && len(paths) > 0 {
				return paths[rand.Intn(len(paths))], volume
			}
		}
	}

	return "", 0
}

func isValidEvent(event string) bool {
	for _, e := range validEvents {
		if e == event {
			return true
		}
	}
	return false
}

func printSoundUsage() {
	fmt.Fprintln(os.Stderr, "Usage: navi sound <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  test <event>   Play the sound for an event")
	fmt.Fprintln(os.Stderr, "  test-all       Play all enabled event sounds")
	fmt.Fprintln(os.Stderr, "  list           List available sound packs")
}
