package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var stdin = bufio.NewReader(os.Stdin)

func confirm(prompt string) bool {
	fmt.Printf("%s ", prompt)
	line, err := stdin.ReadString('\n')
	if err != nil && line == "" {
		fmt.Println()
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func printCandidates(cands []candidate) {
	width := 0
	for _, c := range cands {
		width = max(width, len(c.rule))
	}
	if width < 6 {
		width = 6
	}
	fmt.Printf("\nFound %d candidate%s:\n", len(cands), plural(len(cands)))
	for i, c := range cands {
		fmt.Printf("  %4d  %-*s  %s\n", i+1, width, c.rule, c.path)
	}
	fmt.Println()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func startProgress(counter *atomic.Uint64, label string) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		drawn := 0
		clear := func() {
			if drawn > 0 {
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", drawn))
				drawn = 0
			}
		}
		for {
			select {
			case <-ticker.C:
				clear()
				line := fmt.Sprintf("%s %d directories...", label, counter.Load())
				drawn = len(line)
				fmt.Fprint(os.Stderr, line)
			case <-done:
				clear()
				return
			}
		}
	}()

	return func() {
		close(done)
		wg.Wait()
	}
}
