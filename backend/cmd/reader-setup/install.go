package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const shortcutName = "Keweenaw Reader.lnk"

// DefaultInstallDir is the per-user install root (no admin required).
func DefaultInstallDir() string {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		return filepath.Join(base, "KeweenawReader")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "KeweenawReader"
	}
	return filepath.Join(home, "KeweenawReader")
}

// Install extracts payloadZIP into dest, writes Uninstall.cmd, and creates a desktop shortcut.
func Install(dest string, payloadZIP []byte) error {
	dest = filepath.Clean(dest)
	if dest == "" || dest == "." {
		return fmt.Errorf("invalid install directory")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	if err := extractZip(payloadZIP, dest); err != nil {
		return err
	}
	if err := writeUninstallScript(dest); err != nil {
		return err
	}
	gui := filepath.Join(dest, "reader-gui.exe")
	if _, err := os.Stat(gui); err == nil {
		if err := createDesktopShortcut(gui, dest); err != nil {
			// Non-fatal on headless CI; still report.
			fmt.Fprintf(os.Stderr, "warning: desktop shortcut: %v\n", err)
		}
	}
	return nil
}

func extractZip(payloadZIP []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(payloadZIP), int64(len(payloadZIP)))
	if err != nil {
		return fmt.Errorf("read payload zip: %w", err)
	}
	for _, f := range zr.File {
		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	name := filepath.Clean(f.Name)
	if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
		return fmt.Errorf("refusing unsafe zip path %q", f.Name)
	}
	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("refusing zip path outside dest: %q", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, rc); err != nil {
		return err
	}
	return nil
}

func writeUninstallScript(dest string) error {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", shortcutName)
	body := fmt.Sprintf(`@echo off
echo Removing Keweenaw Reader from:
echo   %s
del /f /q "%s" 2>nul
rmdir /s /q "%s"
echo Done.
pause
`, dest, desktop, dest)
	return os.WriteFile(filepath.Join(dest, "Uninstall.cmd"), []byte(body), 0o644)
}

func createDesktopShortcut(targetEXE, workDir string) error {
	desktop := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
	if desktop == filepath.Join("", "Desktop") || os.Getenv("USERPROFILE") == "" {
		if home, err := os.UserHomeDir(); err == nil {
			desktop = filepath.Join(home, "Desktop")
		}
	}
	lnk := filepath.Join(desktop, shortcutName)
	ps := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$ws = New-Object -ComObject WScript.Shell
$sc = $ws.CreateShortcut('%s')
$sc.TargetPath = '%s'
$sc.WorkingDirectory = '%s'
$sc.Description = 'Keweenaw Endurance race reader'
$sc.Save()
`, escapePS(lnk), escapePS(targetEXE), escapePS(workDir))
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
