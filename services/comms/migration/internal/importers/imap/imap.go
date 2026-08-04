package imap

import (
	"bufio"
	"os"
)

func ImportMailbox(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			out = append(out, line)
		}
	}
	return out, sc.Err()
}
