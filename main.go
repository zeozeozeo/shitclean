package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	workers int
}

type session struct {
	opts    Options
	seen    sync.Map // path -> rule name
	scanned atomic.Uint64
}

func (s *session) scan(root string) []candidate {
	sc := &scanner{
		root: root,
		seen: &s.seen,
		sem:  make(chan struct{}, s.opts.workers),
		dirs: &s.scanned,
	}
	return sc.run()
}

type stats struct {
	total   int
	deleted int
	gone    int
	failed  int
}

func (s *stats) add(o stats) {
	s.total += o.total
	s.deleted += o.deleted
	s.gone += o.gone
	s.failed += o.failed
}

func main() {
	dir := flag.String("dir", ".", "directory to start scanning")
	flag.Parse()

	opts := Options{workers: min(max(runtime.NumCPU()*4, 4), 256)}

	abs, err := filepath.Abs(*dir)
	if err != nil {
		fatalf("invalid -dir: %v", err)
	}
	abs = filepath.Clean(abs)
	if fi, err := os.Stat(abs); err != nil {
		fatalf("cannot access %s: %v", abs, err)
	} else if !fi.IsDir() {
		fatalf("%s is not a directory", abs)
	}

	fmt.Printf("This will recursively search for build directories in %s.\n", abs)
	fmt.Println("You will be prompted before anything is deleted.")
	if !confirm("Continue (y/n)?") {
		return
	}
	fmt.Println()

	sess := &session{opts: opts}
	start := time.Now()
	stop := startProgress(&sess.scanned, "Scanning")
	cands := sess.scan(abs)
	stop()
	fmt.Printf("Scanned %d directories in %s.\n", sess.scanned.Load(), time.Since(start).Round(time.Millisecond))

	if len(cands) == 0 {
		fmt.Println("No build directories found. Good for you.")
		return
	}
	sortCandidates(cands)
	printCandidates(cands)

	declined, st := runRound(cands)
	var total stats
	total.add(st)

	for len(declined) > 0 {
		var next []candidate
		for _, path := range declined {
			if defaultSkip(filepath.Base(path)) {
				continue
			}
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				continue
			}
			fmt.Printf("\nRescanning declined %s...\n", path)
			stop = startProgress(&sess.scanned, "Scanning")
			next = append(next, sess.scan(path)...)
			stop()
		}
		if len(next) == 0 {
			break
		}
		sortCandidates(next)
		printCandidates(next)
		declined, st = runRound(next)
		total.add(st)
	}

	fmt.Println()
	if total.deleted > 0 || total.gone > 0 || total.failed > 0 {
		fmt.Printf("Deleted %d of %d candidates", total.deleted, total.total)
		if total.gone > 0 {
			fmt.Printf(", %d already gone", total.gone)
		}
		if total.failed > 0 {
			fmt.Printf(", %d failed", total.failed)
		}
		fmt.Println(".")
	} else {
		fmt.Println("Do you really think they're this important?")
	}
}

func runRound(cands []candidate) (declined []string, st stats) {
	st.total = len(cands)
	for i, c := range cands {
		if _, err := os.Lstat(c.path); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("  %-8s %s\n", "gone", c.path)
				st.gone++
			} else {
				fmt.Printf("  %-8s %s: %v\n", "failed", c.path, err)
				st.failed++
			}
			continue
		}
		prompt := fmt.Sprintf("(%d/%d) remove %s directory at %s (y/n)?", i+1, len(cands), c.rule, c.path)
		if !confirm(prompt) {
			declined = append(declined, c.path)
			continue
		}
		if err := removeDir(c.path); err != nil {
			fmt.Printf("  %-8s %s: %v\n", "failed", c.path, err)
			st.failed++
			continue
		}
		fmt.Printf("  %-8s %s\n", "removed", c.path)
		st.deleted++
	}
	return declined, st
}

func sortCandidates(c []candidate) {
	slices.SortFunc(c, func(a, b candidate) int { return strings.Compare(a.path, b.path) })
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "shitclean: "+format+"\n", args...)
	os.Exit(1)
}
