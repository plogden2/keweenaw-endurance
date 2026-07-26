package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed payload.zip
var payloadZIP []byte

func main() {
	dest := flag.String("dest", DefaultInstallDir(), "install directory")
	launch := flag.Bool("launch", true, "launch reader-gui after install")
	flag.Parse()

	fmt.Println("Keweenaw Endurance — Reader setup")
	fmt.Printf("Installing to:\n  %s\n\n", *dest)

	if len(payloadZIP) < 22 {
		fmt.Fprintln(os.Stderr, "error: installer payload missing — rebuild with scripts/pack-reader-setup.ps1")
		os.Exit(1)
	}
	if err := Install(*dest, payloadZIP); err != nil {
		fmt.Fprintf(os.Stderr, "install failed: %v\n", err)
		os.Exit(1)
	}

	gui := filepath.Join(*dest, "reader-gui.exe")
	setupTxt := filepath.Join(*dest, "SETUP.txt")
	fmt.Println("Install complete.")
	fmt.Printf("  Reader GUI: %s\n", gui)
	fmt.Printf("  Instructions: %s\n", setupTxt)
	fmt.Println("  Desktop shortcut: Keweenaw Reader")
	fmt.Println()
	fmt.Println("Next: plug in Proxmark (usually COM3), open the Reader GUI,")
	fmt.Println("Test Proxmark, Start bridge, then arm the station on the website.")

	if *launch {
		if _, err := os.Stat(gui); err == nil {
			cmd := exec.Command(gui)
			cmd.Dir = *dest
			_ = cmd.Start()
		}
	}
}
