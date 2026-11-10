package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func runCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func lines(s string) [][]string {
	var res [][]string
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		res = append(res, strings.Fields(l))
	}
	return res
}

func expectLine(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("line = %v, want %v", got, want)
	}
}

func TestSingleFile(t *testing.T) {
	p := writeFile(t, "a.txt", "one two  three\n")
	out, errs, code := runCLI(t, "", p)
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	l := lines(out)
	if len(l) != 1 {
		t.Fatalf("want one line, got %q", out)
	}
	expectLine(t, l[0], "3", p)
}

func TestWhitespaceRuns(t *testing.T) {
	p := writeFile(t, "w.txt", "  a \t\t b\n\n\n c\r\n \t d   \n\n")
	out, _, code := runCLI(t, "", p)
	if code != 0 {
		t.Fatalf("code=%d", code)
	}
	expectLine(t, lines(out)[0], "4", p)
}

func TestEmptyAndWhitespaceOnly(t *testing.T) {
	for _, c := range []string{"", " \n\t \n\n  "} {
		p := writeFile(t, "e.txt", c)
		out, errs, code := runCLI(t, "", p)
		if code != 0 || errs != "" {
			t.Fatalf("code=%d stderr=%q", code, errs)
		}
		l := lines(out)
		if len(l) != 1 {
			t.Fatalf("got %q", out)
		}
		expectLine(t, l[0], "0", p)
	}
}

func TestMultipleFilesTotal(t *testing.T) {
	a := writeFile(t, "a.txt", "one two")
	b := writeFile(t, "b.txt", "")
	c := writeFile(t, "c.txt", "x\ny\nz\n")
	out, errs, code := runCLI(t, "", a, b, c)
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	l := lines(out)
	if len(l) != 4 {
		t.Fatalf("got %q", out)
	}
	expectLine(t, l[0], "2", a)
	expectLine(t, l[1], "0", b)
	expectLine(t, l[2], "3", c)
	expectLine(t, l[3], "5", "total")
}

func TestMissingFile(t *testing.T) {
	a := writeFile(t, "a.txt", "one two")
	c := writeFile(t, "c.txt", "x y z")
	missing := filepath.Join(t.TempDir(), "nope.txt")
	out, errs, code := runCLI(t, "", a, missing, c)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	want := "wordcount: " + missing + ": no such file or directory"
	if strings.TrimSpace(errs) != want {
		t.Errorf("stderr = %q, want %q", errs, want)
	}
	l := lines(out)
	if len(l) != 3 {
		t.Fatalf("got %q", out)
	}
	expectLine(t, l[0], "2", a)
	expectLine(t, l[1], "3", c)
	expectLine(t, l[2], "5", "total")
}

func TestOnlyMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.txt")
	out, errs, code := runCLI(t, "", missing)
	if code != 1 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	if strings.TrimSpace(errs) != "wordcount: "+missing+": no such file or directory" {
		t.Errorf("stderr = %q", errs)
	}
}

func TestDirectoryArgument(t *testing.T) {
	d := t.TempDir()
	_, errs, code := runCLI(t, "", d)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if !strings.HasPrefix(errs, "wordcount: "+d+": ") {
		t.Errorf("stderr = %q", errs)
	}
}

func TestStdin(t *testing.T) {
	out, errs, code := runCLI(t, "hello  big\nworld\t!\n")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d stderr=%q", code, errs)
	}
	if strings.TrimSpace(out) != "4" {
		t.Errorf("stdout = %q", out)
	}
}

func TestStdinEmpty(t *testing.T) {
	out, _, code := runCLI(t, "  \n")
	if code != 0 || strings.TrimSpace(out) != "0" {
		t.Errorf("code=%d out=%q", code, out)
	}
}

func TestStdinError(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(nil, errReader{}, &out, &errb)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if strings.TrimSpace(errb.String()) != "wordcount: stdin: boom" {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestCountWordsAndLongToken(t *testing.T) {
	n, err := countWords(strings.NewReader("a b\n" + strings.Repeat("x", 200000) + " c"))
	if err != nil || n != 4 {
		t.Errorf("n=%d err=%v", n, err)
	}
}
