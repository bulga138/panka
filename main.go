package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/bulga138/panka/config"
	"github.com/bulga138/panka/editor"
	"github.com/bulga138/panka/terminal"
	"github.com/bulga138/panka/version"
)

// Define the command-line flags
var (
	initConfig  = flag.Bool("init-config", false, "Create a default config file and exit.")
	showVersion = flag.Bool("version", false, "Show version information and exit.")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// --- Handle --version flag ---
	if *showVersion {
		fmt.Printf("Panka Editor %s\n", version.GetFullVersion())
		os.Exit(0)
	}

	// --- Handle --init-config flag ---
	if *initConfig {
		cfg := config.DefaultConfig()
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0) // Exit cleanly after creating the file
	}

	// Force terminal reset at startup to ensure clean state
	fmt.Print("\x1b[0m\x1b[2J\x1b[H\x1b[?25h")

	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Set up logging based on config
	if cfg.EnableLogger {
		f, err := os.OpenFile("panka.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f)
		log.Println("--- Panka Editor Started (Logging Enabled) ---")
	} else {
		log.SetOutput(io.Discard)
	}

	log.Printf("Config loaded: %+v", cfg)

	// 3. Parse Arguments
	var filename string
	// Use flag.Args() to get non-flag arguments
	args := flag.Args()
	if len(args) > 1 {
		fmt.Println("Usage: panka [filename]")
		os.Exit(1)
	}
	if len(args) == 1 {
		filename = args[0]
	}
	log.Printf("File to open: %s", filename)

	// 4. Initialize Terminal
	term := terminal.New()
	defer term.Close()

	// 5. Initialize Editor
	e, err := editor.NewEditor(term, cfg, filename)
	if err != nil {
		fmt.Printf("Error initializing editor: %v\n", err)
		log.Fatalf("Error initializing editor: %v", err)
		os.Exit(1)
	}

	// 6. Run the editor
	if err := e.Run(); err != nil {
		fmt.Printf("Error running editor: %v\n", err)
		log.Fatalf("Error running editor: %v", err)
		os.Exit(1)
	}

	log.Println("--- Panka Editor Exited Cleanly ---")
}
