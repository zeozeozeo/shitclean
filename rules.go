package main

import (
	"strings"
)

type rule struct {
	name        string
	files       []string
	dirs        []string
	suffixes    []string
	dirSuffixes []string

	targets []string
	custom  func(*dirInfo) []string

	fileMask      bitset
	dirMask       bitset
	suffixMask    uint64
	dirSuffixMask uint64
}

const bitsetWords = 8

type bitset [bitsetWords]uint64

func (b *bitset) set(id int) { b[id>>6] |= 1 << (uint(id) & 63) }

func (b *bitset) has(id int) bool { return b[id>>6]&(1<<(uint(id)&63)) != 0 }

func (b bitset) intersects(o bitset) bool {
	for i := range b {
		if b[i]&o[i] != 0 {
			return true
		}
	}
	return false
}

var (
	markerIDs    = map[string]int{}
	suffixIDs    = map[string]int{}
	dirSuffixIDs = map[string]int{}

	fileSuffixTable [maxSuffixLen][]suffixEntry
	dirSuffixTable  [maxSuffixLen][]suffixEntry

	allRules   []*rule
	selfRules  []*rule
	childRules []*rule
)

type suffixEntry struct {
	suffix []byte
	bit    uint64
}

const maxSuffixLen = 32

func markerID(name string) int {
	if id, ok := markerIDs[name]; ok {
		return id
	}
	id := len(markerIDs)
	if id >= bitsetWords*64 {
		panic("shitclean: too many distinct markers")
	}
	markerIDs[name] = id
	return id
}

func registerSuffix(table *[maxSuffixLen][]suffixEntry, ids map[string]int, suffix, what string) uint64 {
	suffix = strings.ToLower(suffix)
	if id, ok := ids[suffix]; ok {
		return 1 << uint(id)
	}
	id := len(ids)
	if id >= 64 {
		panic("shitclean: too many distinct " + what + " suffixes")
	}
	ids[suffix] = id
	if len(suffix) < maxSuffixLen {
		table[len(suffix)] = append(table[len(suffix)], suffixEntry{
			suffix: []byte(suffix),
			bit:    1 << uint(id),
		})
	}
	return 1 << uint(id)
}

func compileRules() {
	seen := map[string]bool{}
	for _, r := range allRules {
		if seen[r.name] {
			panic("shitclean: duplicate rule name " + r.name)
		}
		seen[r.name] = true

		for _, n := range r.files {
			r.fileMask.set(markerID(n))
		}
		for _, n := range r.dirs {
			r.dirMask.set(markerID(n))
		}
		for _, s := range r.suffixes {
			r.suffixMask |= registerSuffix(&fileSuffixTable, suffixIDs, s, "file")
		}
		for _, s := range r.dirSuffixes {
			r.dirSuffixMask |= registerSuffix(&dirSuffixTable, dirSuffixIDs, s, "directory")
		}
		selfOnly := len(r.targets) > 0
		for _, t := range r.targets {
			if t == "" {
				selfRules = append(selfRules, r)
				continue
			}
			selfOnly = false
			base := t
			if i := strings.LastIndexByte(base, '/'); i >= 0 {
				base = base[i+1:]
			}
			if base != "" {
				markerID(base)
			}
		}
		if !selfOnly {
			childRules = append(childRules, r)
		}
	}
}

func (r *rule) matches(d *dirInfo) bool {
	if r.fileMask != (bitset{}) && !d.files.intersects(r.fileMask) {
		return false
	}
	if r.dirMask != (bitset{}) && !d.dirs.intersects(r.dirMask) {
		return false
	}
	if r.suffixMask != 0 && d.suffixMask&r.suffixMask == 0 {
		return false
	}
	if r.dirSuffixMask != 0 && d.dirSuffixMask&r.dirSuffixMask == 0 {
		return false
	}
	return true
}

func (r *rule) hit(d *dirInfo) ([]string, bool) {
	if !r.matches(d) {
		return nil, false
	}
	if r.custom != nil {
		return r.custom(d), true
	}
	if len(r.targets) == 0 {
		var out []string
		for _, name := range d.dirSuffixHits {
			if i := strings.LastIndexByte(name, '.'); i > 0 {
				if id, ok := dirSuffixIDs[strings.ToLower(name[i:])]; ok && r.dirSuffixMask&(1<<uint(id)) != 0 {
					out = append(out, name)
				}
			}
		}
		return out, true
	}
	return r.targets, true
}
