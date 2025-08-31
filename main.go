package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxRecursionDepth = 1000

var (
	dirCount  uint64
	skipDirs  = map[string]bool{"target": true, "node_modules": true, "CMakeFiles": true}
	detectors = map[string]func(string) (bool, string){
		"cargo": detectCargo,
		"node":  detectNode,
		"cmake": detectCMake,
	}
)

type foundDir struct {
	path string
	typ  string
}

func detectCargo(path string) (bool, string) {
	target := filepath.Join(path, "target")
	_, err := os.Stat(filepath.Join(path, "Cargo.toml"))
	if err != nil {
		return false, ""
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return false, ""
	}
	entries, _ := os.ReadDir(target)
	return len(entries) > 0, target
}

func detectNode(path string) (bool, string) {
	nodeModules := filepath.Join(path, "node_modules")
	_, err := os.Stat(filepath.Join(path, "package.json"))
	if err != nil {
		return false, ""
	}
	info, err := os.Stat(nodeModules)
	if err != nil || !info.IsDir() {
		return false, ""
	}
	entries, _ := os.ReadDir(nodeModules)
	return len(entries) > 0, nodeModules
}

func detectCMake(path string) (bool, string) {
	_, err1 := os.Stat(filepath.Join(path, "CMakeCache.txt"))
	info, err2 := os.Stat(filepath.Join(path, "CMakeFiles"))
	if err1 != nil || err2 != nil || !info.IsDir() {
		return false, ""
	}
	entries, _ := os.ReadDir(filepath.Join(path, "CMakeFiles"))
	return len(entries) > 0, path
}

func walkDir(path string, depth int, sem chan struct{}, wg *sync.WaitGroup, results chan<- foundDir) {
	defer wg.Done()

	if depth >= maxRecursionDepth {
		return
	}

	for typ, detector := range detectors {
		if found, dirPath := detector(path); found {
			results <- foundDir{path: dirPath, typ: typ}
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || skipDirs[entry.Name()] {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())
		if isSymlink(fullPath) {
			continue
		}

		wg.Add(1)
		select {
		case sem <- struct{}{}:
			go walkDir(fullPath, depth+1, sem, wg, results)
		default:
			walkDir(fullPath, depth+1, sem, wg, results)
		}
	}
	atomic.AddUint64(&dirCount, 1)
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

func confirm(prompt string) bool {
	fmt.Print(prompt + " ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y"
}

func printProgress(stopChan chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count := atomic.LoadUint64(&dirCount)
			fmt.Printf("\rChecked %d directories...", count)
		case <-stopChan:
			fmt.Printf("\rChecked %d directories...\n", atomic.LoadUint64(&dirCount))
			return
		}
	}
}

func main() {
	startDir := flag.String("dir", ".", "directory to start cleaning")
	flag.Parse()

	absPath, _ := filepath.Abs(*startDir)
	if !confirm(fmt.Sprintf("This will recursively search build folders in %s. You will be prompted to delete each one. Are you sure (y/n)?", absPath)) {
		return
	}

	sem := make(chan struct{}, 50)
	var wg sync.WaitGroup
	results := make(chan foundDir, 100)

	stopProgress := make(chan struct{})
	var progressWg sync.WaitGroup
	progressWg.Add(1)
	go printProgress(stopProgress, &progressWg)

	wg.Add(1)
	go walkDir(*startDir, 0, sem, &wg, results)

	go func() {
		wg.Wait()
		close(results)
	}()

	var found []foundDir
	for fd := range results {
		found = append(found, fd)
		fmt.Printf("\rFound %s (type: %s)\n", fd.path, fd.typ)
	}

	close(stopProgress)
	progressWg.Wait()

	totalDirs := atomic.LoadUint64(&dirCount)
	fmt.Printf("\nProcessed %d directories, found %d candidates.\n", totalDirs, len(found))
	if len(found) == 0 {
		fmt.Println("Good for you.")
		return
	}

	deleted := 0
	for i, fd := range found {
		if confirm(fmt.Sprintf("(%d/%d) remove %s directory at %s (y/n)?", i+1, len(found), fd.typ, fd.path)) {
			if err := os.RemoveAll(fd.path); err == nil {
				deleted++
			} else {
				fmt.Printf("Error removing %s: %v\n", fd.path, err)
			}
		}
	}

	if deleted > 0 {
		fmt.Printf("Eliminated %d disk space abusers.\n", deleted)
	} else {
		fmt.Println("Do you really think they're this important?")
	}
}
