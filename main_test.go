package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Fails the test if clang-format is not on PATH.
func requireClangFormat(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("clang-format"); err != nil {
		t.Fatalf("clang-format not on PATH: %v", err)
	}
}

// writeClangFormatConfig drops a minimal, deterministic .clang-format into dir
// so test output does not vary with the user's installed clang-format defaults.
func writeClangFormatConfig(t *testing.T, dir string) {
	t.Helper()
	cfg := strings.Join([]string{
		"BasedOnStyle: LLVM",
		"IndentWidth: 4",
		"ColumnLimit: 100",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, ".clang-format"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestInjectPlaceholders(t *testing.T) {
	in := strings.Join([]string{
		"%module mylib",
		"",
		"%{",
		"#include \"mylib.h\"",
		"%}",
		"",
		"    %include \"typemaps.i\"", // indented directive must be matched
		"int x;",
	}, "\n")

	out, directives := injectPlaceholders(in, "TAG01")
	if len(directives) != 4 {
		t.Fatalf("expected 4 directives, got %d: %v", len(directives), directives)
	}
	for _, want := range []string{
		"#pragma SWIG_TAG01_0",
		"#pragma SWIG_TAG01_1",
		"#pragma SWIG_TAG01_2",
		"#pragma SWIG_TAG01_3",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "%module") || strings.Contains(out, "%include") {
		t.Errorf("output still contains raw %% directives:\n%s", out)
	}
	if !strings.Contains(out, "#include \"mylib.h\"") {
		t.Errorf("non-directive line was modified:\n%s", out)
	}
}

func TestRestorePlaceholdersRoundTrip(t *testing.T) {
	in := strings.Join([]string{
		"%module mylib",
		"%{",
		"#include \"mylib.h\"",
		"%}",
		"    %include \"typemaps.i\"",
		"int x;",
	}, "\n")

	guarded, directives := injectPlaceholders(in, "ABC123")
	got := restorePlaceholders(guarded, "ABC123", directives)
	if got != in {
		t.Errorf("round-trip mismatch\nwant:\n%s\ngot:\n%s", in, got)
	}
}

func TestRestoreOnlyMatchesOurTag(t *testing.T) {
	// A pragma with a different tag must be left alone.
	const foreign = "#pragma SWIG_FFFFFF_0"
	guarded, directives := injectPlaceholders("int x;\n%module mylib", "ABC123")
	withForeign := guarded + "\n" + foreign
	got := restorePlaceholders(withForeign, "ABC123", directives)
	if !strings.Contains(got, foreign) {
		t.Errorf("foreign pragma was incorrectly replaced:\n%s", got)
	}
}

func TestRestoreOutOfRangeIndexIsLeftAlone(t *testing.T) {
	// Simulate clang-format somehow producing a bogus higher index.
	const bogus = "#pragma SWIG_ABC123_999"
	got := restorePlaceholders(bogus, "ABC123", []string{"%module mylib"})
	if got != bogus {
		t.Errorf("expected bogus index to be preserved, got: %q", got)
	}
}

func TestFormatFileChangesAndIsIdempotent(t *testing.T) {
	requireClangFormat(t)
	dir := t.TempDir()
	writeClangFormatConfig(t, dir)

	src := strings.Join([]string{
		"%module   mylib",
		"",
		"%{",
		"#include \"mylib.h\"",
		"%}",
		"",
		"int  add(int   a,int b);",
		"",
		"%inline %{",
		"int multiply(int a,int b){return a*b;}",
		"%}",
		"",
	}, "\n")
	path := filepath.Join(dir, "sample.i")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := formatFile(path, false)
	if err != nil {
		t.Fatalf("first format failed: %v", err)
	}
	if !changed {
		t.Fatal("expected first run to report a change")
	}

	formatted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(formatted)

	// SWIG directives must be preserved verbatim.
	for _, want := range []string{"%module   mylib", "%{", "%}", "%inline %{"} {
		if !strings.Contains(body, want) {
			t.Errorf("SWIG directive %q was lost:\n%s", want, body)
		}
	}

	// No placeholder pragma should leak into the output.
	if strings.Contains(body, "#pragma SWIG_") {
		t.Errorf("placeholder leaked into output:\n%s", body)
	}

	// Second run is a no-op.
	changed2, err := formatFile(path, false)
	if err != nil {
		t.Fatalf("second format failed: %v", err)
	}
	if changed2 {
		t.Errorf("expected idempotent second run; output:\n%s", body)
	}
}

func TestFormatFileCheckModeDoesNotWrite(t *testing.T) {
	requireClangFormat(t)
	dir := t.TempDir()
	writeClangFormatConfig(t, dir)

	src := "%module mylib\nint  x;\n"
	path := filepath.Join(dir, "sample.i")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := formatFile(path, true)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}
	if !changed {
		t.Fatal("expected --check to report changed=true")
	}
	got, _ := os.ReadFile(path)
	if string(got) != src {
		t.Errorf("--check must not modify the file; got:\n%s", got)
	}
}

func TestFormatFilePreservesCRLF(t *testing.T) {
	requireClangFormat(t)
	dir := t.TempDir()
	writeClangFormatConfig(t, dir)

	src := "%module mylib\r\nint  x;\r\n"
	path := filepath.Join(dir, "sample.i")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := formatFile(path, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Contains(got, []byte("\r\n")) {
		t.Errorf("CRLF line endings were not preserved: %q", got)
	}
	if bytes.Contains(bytes.ReplaceAll(got, []byte("\r\n"), []byte{}), []byte("\n")) {
		t.Errorf("output mixes CRLF and LF: %q", got)
	}
}

func TestFormatFilePreservesPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions test")
	}
	requireClangFormat(t)
	dir := t.TempDir()
	writeClangFormatConfig(t, dir)

	src := "%module mylib\nint  x;\n"
	path := filepath.Join(dir, "sample.i")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := formatFile(path, false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("permissions not preserved: want 0600, got %o", got)
	}
}
