package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"unicode"
)

// countWords counts whitespace-separated words in r.
func countWords(r io.Reader) (int, error) {
	br := bufio.NewReader(r)
	count, inWord := 0, false
	for {
		c, _, err := br.ReadRune()
		if err == io.EOF {
			return count, nil
		}
		if err != nil {
			return 0, err
		}
		if unicode.IsSpace(c) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
}

func countFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return countWords(f)
}

// reason returns the underlying error text, without the "op path:" prefix.
func reason(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		n, err := countWords(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "wordcount: stdin: %s\n", reason(err))
			return 1
		}
		fmt.Fprintln(stdout, n)
		return 0
	}
	code, total := 0, 0
	for _, path := range args {
		n, err := countFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "wordcount: %s: %s\n", path, reason(err))
			code = 1
			continue
		}
		total += n
		fmt.Fprintf(stdout, "%d %s\n", n, path)
	}
	if len(args) > 1 {
		fmt.Fprintf(stdout, "%d total\n", total)
	}
	return code
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
