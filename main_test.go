package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func runArgs(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), errb.String(), code
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSingleFile(t *testing.T) {
	p := writeTemp(t, "a.txt", "one two three\n")
	out, errs, code := runArgs(t, "", p)
	if out != "3 "+p+"\n" || errs != "" || code != 0 {
		t.Fatalf("got %q %q %d", out, errs, code)
	}
}

func TestWordCounts(t *testing.T) {
	cases := map[string]int{
		"":                        0,
		"   \n\t \n":              0,
		"a  b\t\tc\n\n\nd":        4,
		"  lead and trail  \n":    3,
		"one":                     1,
		"héllo wörld 日本 語":        4,
		"a\u00a0b":                2,
		"a\u3000b\u3000c":         3,
		strings.Repeat("x", 1<<20): 1,
	}
	for content, want := range cases {
		p := writeTemp(t, "f.txt", content)
		out, _, code := runArgs(t, "", p)
		w := want
		if code != 0 || out != strconv.Itoa(w)+" "+p+"\n" {
			t.Errorf("content %.20q: got %q code %d, want count %d", content, out, code, w)
		}
	}
}

func TestMultipleFilesTotal(t *testing.T) {
	a := writeTemp(t, "a.txt", "a b")
	b := writeTemp(t, "b.txt", "")
	c := writeTemp(t, "c.txt", "x y z\n")
	out, errs, code := runArgs(t, "", a, b, c)
	want := "2 " + a + "\n0 " + b + "\n3 " + c + "\n5 total\n"
	if out != want || errs != "" || code != 0 {
		t.Fatalf("got %q %q %d, want %q", out, errs, code, want)
	}
}

func TestMissingFile(t *testing.T) {
	a := writeTemp(t, "a.txt", "a b")
	missing := filepath.Join(t.TempDir(), "nope.txt")
	c := writeTemp(t, "c.txt", "x y z")
	out, errs, code := runArgs(t, "", a, missing, c)
	if code != 1 {
		t.Fatalf("code %d", code)
	}
	if want := "2 " + a + "\n3 " + c + "\n5 total\n"; out != want {
		t.Fatalf("stdout %q want %q", out, want)
	}
	if want := "wordcount: " + missing + ": no such file or directory\n"; errs != want {
		t.Fatalf("stderr %q want %q", errs, want)
	}
}

func TestSingleMissingFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.txt")
	out, errs, code := runArgs(t, "", missing)
	if code != 1 || out != "" || errs != "wordcount: "+missing+": no such file or directory\n" {
		t.Fatalf("got %q %q %d", out, errs, code)
	}
}

func TestStdin(t *testing.T) {
	out, errs, code := runArgs(t, "héllo  wörld\n日本\t語 ")
	if out != "4\n" || errs != "" || code != 0 {
		t.Fatalf("got %q %q %d", out, errs, code)
	}
	out, _, code = runArgs(t, "")
	if out != "0\n" || code != 0 {
		t.Fatalf("empty stdin: %q %d", out, code)
	}
}

func TestStdinError(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(nil, errReader{}, &out, &errb)
	if code != 1 || errb.String() != "wordcount: stdin: boom\n" {
		t.Fatalf("got %q %d", errb.String(), code)
	}
}
