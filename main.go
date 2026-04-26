package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// version is set by goreleaser via -ldflags "-X main.version=..."
var version = "dev"

// Matches the placeholder lines we inject, allowing clang-format to add leading
// indentation without breaking the round-trip back to the original directive.
var placeholderRe = regexp.MustCompile(`[ \t]*#pragma SWIG_([0-9A-F]+)_(\d+)`)

const usage = `Usage: clang-format-swig [OPTIONS] FILE...

Format SWIG (.i) files in place using clang-format. Lines starting with %
(SWIG directives) are preserved verbatim while the surrounding C/C++ is
formatted normally. Honors the nearest .clang-format config.

Options:
  --check     Exit non-zero if any file would be reformatted; do not write
  --version   Print version and exit
  --help      Show this help

Exit status: 0 if nothing changed, 1 if any file was (or would be) reformatted.`

func makeTag() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return strings.ToUpper(hex.EncodeToString(b[:]))
}

// injectPlaceholders replaces every %-directive line with a #pragma placeholder
// that clang-format preserves verbatim and that has no effect on brace depth.
func injectPlaceholders(content, tag string) (string, []string) {
	lines := strings.Split(content, "\n")
	var directives []string

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "%") {
			lines[i] = fmt.Sprintf("#pragma SWIG_%s_%d", tag, len(directives))
			directives = append(directives, line)
		}
	}

	return strings.Join(lines, "\n"), directives
}

func restorePlaceholders(content, tag string, directives []string) string {
	return placeholderRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := placeholderRe.FindStringSubmatch(match)
		if sub[1] != tag {
			return match
		}
		idx, err := strconv.Atoi(sub[2])
		if err != nil || idx >= len(directives) {
			return match
		}
		return directives[idx]
	})
}

func formatFile(path string, check bool) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	hasCRLF := bytes.Contains(raw, []byte("\r\n"))
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")

	tag := makeTag()
	guarded, directives := injectPlaceholders(content, tag)

	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	cmd := exec.Command("clang-format", "--assume-filename="+stem+".cpp", "-")
	cmd.Stdin = strings.NewReader(guarded)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return false, fmt.Errorf("clang-format: %w", err)
		}
		return false, fmt.Errorf("clang-format: %s", msg)
	}

	result := restorePlaceholders(stdout.String(), tag, directives)
	if hasCRLF {
		result = strings.ReplaceAll(result, "\n", "\r\n")
	}

	out := []byte(result)
	if bytes.Equal(out, raw) {
		return false, nil
	}

	if !check {
		if err := os.WriteFile(path, out, info.Mode().Perm()); err != nil {
			return false, err
		}
	}

	return true, nil
}

func main() {
	check := false
	endOfOpts := false
	files := make([]string, 0, len(os.Args))

	for _, arg := range os.Args[1:] {
		if endOfOpts {
			files = append(files, arg)
			continue
		}
		switch arg {
		case "--":
			endOfOpts = true
		case "--check":
			check = true
		case "--version", "-v":
			fmt.Println(version)
			return
		case "--help", "-h":
			fmt.Println(usage)
			return
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "clang-format-swig: unknown flag: %s\n", arg)
				os.Exit(2)
			}
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	changed := false
	for _, f := range files {
		ok, err := formatFile(f, check)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", f, err)
			os.Exit(1)
		}
		if ok {
			changed = true
			if check {
				fmt.Printf("would reformat %s\n", f)
			} else {
				fmt.Printf("reformatted %s\n", f)
			}
		}
	}

	if changed {
		os.Exit(1)
	}
}
