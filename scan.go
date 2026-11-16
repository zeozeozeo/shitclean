package main

import (
	"sync"
	"sync/atomic"
)

const maxDepth = 512

type candidate struct {
	path string
	rule string
}

type scanner struct {
	root string
	seen *sync.Map
	sem  chan struct{}
	wg   sync.WaitGroup
	mu   sync.Mutex
	out  []candidate
	dirs *atomic.Uint64
}

func (s *scanner) run() []candidate {
	s.wg.Add(1)
	s.walk(s.root, 0)
	s.wg.Wait()
	return s.out
}

func (s *scanner) walk(path string, depth int) {
	defer s.wg.Done()
	if depth > maxDepth {
		return
	}

	info, err := readDirInfo(path)
	if err != nil {
		return
	}
	s.dirs.Add(1)

	if s.inspect(info) {
		return
	}

	for _, name := range info.dirNames {
		if defaultSkip(name) {
			continue
		}
		child := joinPath(path, name)
		if _, claimed := s.seen.Load(child); claimed {
			continue
		}

		s.wg.Add(1)
		select {
		case s.sem <- struct{}{}:
			go func() {
				defer func() { <-s.sem }()
				s.walk(child, depth+1)
			}()
		default:
			s.walk(child, depth+1)
		}
	}
}

func (s *scanner) inspect(d *dirInfo) bool {
	for _, r := range selfRules {
		if !r.matches(d) {
			continue
		}
		if s.claim(d.path, r.name) {
			return true
		}
	}

	for _, r := range childRules {
		targets, ok := r.hit(d)
		if !ok {
			continue
		}
		for _, t := range targets {
			p, ok := d.target(t)
			if !ok || !dirNonEmpty(p) {
				continue
			}
			s.claim(p, r.name)
		}
	}
	return false
}

func (s *scanner) claim(path, ruleName string) bool {
	if path == s.root {
		return false
	}
	if _, loaded := s.seen.LoadOrStore(path, ruleName); loaded {
		return false
	}
	s.mu.Lock()
	s.out = append(s.out, candidate{path: path, rule: ruleName})
	s.mu.Unlock()
	return true
}
