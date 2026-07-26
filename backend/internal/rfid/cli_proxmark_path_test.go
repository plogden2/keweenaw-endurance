package rfid

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProxmarkRuntimeBinSideBySide(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DLL layout only")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "proxmark3.exe")
	if err := os.WriteFile(cli, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := proxmarkRuntimeBin(cli); got != "" {
		t.Fatalf("expected empty without side-by-side runtime, got %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "libgcc_s_seh-1.dll"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := proxmarkRuntimeBin(cli); got != dir {
		t.Fatalf("side-by-side: got %q want %q", got, dir)
	}
}

func TestProxmarkRuntimeBinPlatformsDir(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows DLL layout only")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "proxmark3.exe")
	if err := os.WriteFile(cli, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "platforms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := proxmarkRuntimeBin(cli); got != dir {
		t.Fatalf("platforms dir: got %q want %q", got, dir)
	}
}

func TestResolveProxmarkExecutableFromWrapper(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "proxmark3.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdPath := filepath.Join(dir, "pm3.cmd")
	root := filepath.Join(dir, "root")
	if err := os.MkdirAll(filepath.Join(root, "pm3", "proxmark3", "client"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "pm3", "proxmark3", "client", "proxmark3.exe")
	if err := os.WriteFile(nested, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := "@echo off\r\nset \"PM3_ROOT=" + root + "\"\r\nset \"PM3_EXE=%PM3_ROOT%\\pm3\\proxmark3\\client\\proxmark3.exe\"\r\n"
	if err := os.WriteFile(cmdPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := resolveProxmarkExecutable(cmdPath)
	if got != nested {
		t.Fatalf("got %q want %q", got, nested)
	}
	_ = exe
	if got := resolveProxmarkExecutable(exe); got != exe {
		t.Fatalf("exe passthrough: got %q", got)
	}
}

func TestWithEnvVar(t *testing.T) {
	env := []string{"FOO=1", "PATH=C:\\Windows"}
	got := withEnvVar(env, "QT_PLUGIN_PATH", `C:\app`)
	found := false
	for _, e := range got {
		if e == `QT_PLUGIN_PATH=C:\app` {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing QT_PLUGIN_PATH in %#v", got)
	}
	got2 := withEnvVar(got, "QT_PLUGIN_PATH", `C:\other`)
	count := 0
	for _, e := range got2 {
		if len(e) >= 15 && e[:15] == "QT_PLUGIN_PATH=" {
			count++
			if e != `QT_PLUGIN_PATH=C:\other` {
				t.Fatalf("want replace, got %q", e)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected one QT_PLUGIN_PATH, got %d in %#v", count, got2)
	}
}
