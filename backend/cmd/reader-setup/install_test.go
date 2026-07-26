package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractZipSafe(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("SETUP.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	w2, err := zw.Create("proxmark/readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w2.Write([]byte("pm3")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := extractZip(buf.Bytes(), dest); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dest, "SETUP.txt"))
	if err != nil || string(raw) != "hello" {
		t.Fatalf("SETUP.txt: %v %q", err, raw)
	}
	if _, err := os.Stat(filepath.Join(dest, "proxmark", "readme.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(`..\evil.txt`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("x"))
	_ = zw.Close()
	if err := extractZip(buf.Bytes(), t.TempDir()); err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestInstallWritesUninstall(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("SETUP.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte("setup"))
	_ = zw.Close()

	dest := filepath.Join(t.TempDir(), "KeweenawReader")
	if err := Install(dest, buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "Uninstall.cmd")); err != nil {
		t.Fatal(err)
	}
}
