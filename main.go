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

const (
	maxRecursionDepth = 1000
	concurrencyLimit  = 50
)

var (
	dirCount uint64
	skipDirs = map[string]bool{
		"target":       true,
		"node_modules": true,
		"CMakeFiles":   true,
		"build":        true,
		"bin":          true,
		"obj":          true,
		"dist":         true,
		".gradle":      true,
		".idea":        true,
		".vscode":      true,
		".dub":         true,
		".build":       true,
	}
)

// dirEntriesCache caches the results of os.ReadDir for already read paths
var dirEntriesCache sync.Map

// readDirCached reads a directory once and caches the result
func readDirCached(path string) ([]os.DirEntry, error) {
	if v, ok := dirEntriesCache.Load(path); ok {
		return v.([]os.DirEntry), nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	dirEntriesCache.Store(path, entries)
	return entries, nil
}

// return (found, directoryToDelete)
type detectorFunc func(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string)

var detectors = map[string]detectorFunc{
	"cargo":    detectCargo,
	"node":     detectNode,
	"cmake":    detectCMake,
	"maven":    detectMaven,
	"gradle":   detectGradle,
	"dotnet":   detectDotNet,
	"python":   detectPython,
	"d":        detectD,
	"jai":      detectJai,
	"swiftpm":  detectSwiftPM,
	"qobs":     detectQobs,
	"bazel":    detectBazel,
	"meson":    detectMeson,
	"ninja":    detectNinja,
	"sbt":      detectSBT,
	"cabal":    detectCabal,
	"stack":    detectStack,
	"composer": detectComposer,
	"bundler":  detectBundler,
	"pnpm":     detectPNPM,
	"bun":      detectBun,
	"expo":     detectExpo,
	"next":     detectNextJS,
	"angular":  detectAngular,
	"unreal":   detectUnreal,
	"unity":    detectUnity,
	"android":  detectAndroid,
	"flutter":  detectFlutter,
	"mix":      detectMix,
	"rebar":    detectRebar,
}

//
// helpers
//

func hasName(nameMap map[string]os.DirEntry, name string) bool {
	_, ok := nameMap[name]
	return ok
}

func anySuffix(entries []os.DirEntry, suf string) bool {
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), suf) {
			return true
		}
	}
	return false
}

func nonEmptyDir(path string) bool {
	entries, err := readDirCached(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

//
// detectors
//

func detectCargo(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "Cargo.toml") {
		return false, ""
	}
	target := filepath.Join(path, "target")
	if nonEmptyDir(target) {
		return true, target
	}
	return false, ""
}

func detectQobs(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "Qobs.toml") {
		return false, ""
	}
	build := filepath.Join(path, "build")
	if nonEmptyDir(build) {
		return true, build
	}
	return false, ""
}

func detectNode(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "package.json") {
		return false, ""
	}
	nodeModules := filepath.Join(path, "node_modules")
	if nonEmptyDir(nodeModules) {
		return true, nodeModules
	}
	return false, ""
}

func detectCMake(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "CMakeCache.txt") || !hasName(nameMap, "CMakeFiles") {
		return false, ""
	}
	cmakeFilesPath := filepath.Join(path, "CMakeFiles")
	if nonEmptyDir(cmakeFilesPath) {
		return true, path
	}
	return false, ""
}

func detectMaven(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "pom.xml") {
		return false, ""
	}
	target := filepath.Join(path, "target")
	if nonEmptyDir(target) {
		return true, target
	}
	return false, ""
}

func detectGradle(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	// build.gradle or build.gradle.kts
	if !hasName(nameMap, "build.gradle") && !hasName(nameMap, "build.gradle.kts") {
		return false, ""
	}
	buildDir := filepath.Join(path, "build")
	if nonEmptyDir(buildDir) {
		return true, buildDir
	}
	return false, ""
}

func detectDotNet(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	// check bin or obj dir presence and non-empty, and .csproj or .sln exists in current dir
	foundProj := false
	for _, e := range entries {
		if !e.IsDir() {
			l := strings.ToLower(e.Name())
			if strings.HasSuffix(l, ".csproj") || strings.HasSuffix(l, ".sln") {
				foundProj = true
				break
			}
		}
	}
	if !foundProj {
		return false, ""
	}
	// prefer bin then obj
	binDir := filepath.Join(path, "bin")
	if nonEmptyDir(binDir) {
		return true, binDir
	}
	objDir := filepath.Join(path, "obj")
	if nonEmptyDir(objDir) {
		return true, objDir
	}
	return false, ""
}

func detectPython(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	// need either setup.py or pyproject.toml
	if !hasName(nameMap, "setup.py") && !hasName(nameMap, "pyproject.toml") {
		return false, ""
	}
	buildDir := filepath.Join(path, "build")
	if nonEmptyDir(buildDir) {
		return true, buildDir
	}
	distDir := filepath.Join(path, "dist")
	if nonEmptyDir(distDir) {
		return true, distDir
	}
	return false, ""
}

func detectD(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	// need dub.json or dub.sdl and a .dub directory non-empty
	if !hasName(nameMap, "dub.json") && !hasName(nameMap, "dub.sdl") {
		return false, ""
	}
	dubDir := filepath.Join(path, ".dub")
	if nonEmptyDir(dubDir) {
		return true, dubDir
	}
	return false, ""
}

func detectJai(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	// check for any *.jai file in current dir
	if !anySuffix(entries, ".jai") {
		return false, ""
	}
	// check possible build dirs
	candidates := []string{filepath.Join(path, "bin"), filepath.Join(path, ".build")}
	for _, d := range candidates {
		if nonEmptyDir(d) {
			return true, d
		}
	}
	return false, ""
}

func detectSwiftPM(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "Package.swift") {
		return false, ""
	}
	p := filepath.Join(path, ".build")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectBazel(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "WORKSPACE") && !hasName(nameMap, "BUILD") {
		return false, ""
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bazel-") && nonEmptyDir(filepath.Join(path, e.Name())) {
			return true, filepath.Join(path, e.Name())
		}
	}
	return false, ""
}

func detectMeson(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "meson.build") {
		return false, ""
	}
	for _, d := range []string{"build", "_build"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectNinja(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "build.ninja") {
		return false, ""
	}
	p := filepath.Join(path, "build")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectSBT(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "build.sbt") {
		return false, ""
	}
	p := filepath.Join(path, "target")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectCabal(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	for name := range nameMap {
		if strings.HasSuffix(name, ".cabal") {
			for _, d := range []string{"dist-newstyle", "dist"} {
				p := filepath.Join(path, d)
				if nonEmptyDir(p) {
					return true, p
				}
			}
		}
	}
	return false, ""
}

func detectStack(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "stack.yaml") {
		return false, ""
	}
	p := filepath.Join(path, ".stack-work")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectComposer(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "composer.json") {
		return false, ""
	}
	p := filepath.Join(path, "vendor")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectBundler(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "Gemfile") {
		return false, ""
	}
	p := filepath.Join(path, "vendor", "bundle")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectPNPM(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "pnpm-lock.yaml") {
		return false, ""
	}
	for _, d := range []string{"node_modules", ".pnpm-store"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectBun(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "bun.lockb") {
		return false, ""
	}
	for _, d := range []string{"node_modules", ".bun"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectExpo(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "app.json") && !hasName(nameMap, "app.config.js") {
		return false, ""
	}
	for _, d := range []string{".expo", ".expo-shared"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectNextJS(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "package.json") {
		return false, ""
	}
	p := filepath.Join(path, ".next")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectAngular(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "angular.json") {
		return false, ""
	}
	p := filepath.Join(path, "dist")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectUnreal(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	for name := range nameMap {
		if strings.HasSuffix(name, ".uproject") {
			for _, d := range []string{"Intermediate", "Saved", "Binaries"} {
				p := filepath.Join(path, d)
				if nonEmptyDir(p) {
					return true, p
				}
			}
		}
	}
	return false, ""
}

func detectUnity(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "ProjectSettings") {
		return false, ""
	}
	for _, d := range []string{"Library", "Temp", "Logs", "obj"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectAndroid(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "AndroidManifest.xml") {
		return false, ""
	}
	p := filepath.Join(path, "build")
	if nonEmptyDir(p) {
		return true, p
	}
	return false, ""
}

func detectFlutter(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "pubspec.yaml") {
		return false, ""
	}
	for _, d := range []string{"build", ".dart_tool"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectMix(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "mix.exs") {
		return false, ""
	}
	for _, d := range []string{"_build", "deps"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func detectRebar(path string, entries []os.DirEntry, nameMap map[string]os.DirEntry) (bool, string) {
	if !hasName(nameMap, "rebar.config") {
		return false, ""
	}
	for _, d := range []string{"_build", "deps"} {
		p := filepath.Join(path, d)
		if nonEmptyDir(p) {
			return true, p
		}
	}
	return false, ""
}

func walkDir(path string, depth int, sem chan struct{}, wg *sync.WaitGroup, results chan<- foundDir) {
	defer wg.Done()

	if depth >= maxRecursionDepth {
		return
	}

	entries, err := readDirCached(path)
	if err != nil {
		return
	}

	// for existence checks
	nameMap := make(map[string]os.DirEntry, len(entries))
	for _, e := range entries {
		nameMap[e.Name()] = e
	}

	// run detectors using entries+map
	for typ, detector := range detectors {
		if found, dirPath := detector(path, entries, nameMap); found {
			results <- foundDir{path: dirPath, typ: typ}
		}
	}

	// iterate/skip child directories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if skipDirs[name] {
			continue
		}

		fullPath := filepath.Join(path, name)
		// skip symlinks
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		wg.Add(1)
		// try to acquire semaphore slot and launch goroutine that will release slot when done
		select {
		case sem <- struct{}{}:
			go func(p string) {
				// run walker
				walkDir(p, depth+1, sem, wg, results)
				<-sem
			}(fullPath)
		default:
			// no semaphore slot available, run in this goroutine
			walkDir(fullPath, depth+1, sem, wg, results)
		}
	}

	atomic.AddUint64(&dirCount, 1)
}

type foundDir struct {
	path string
	typ  string
}

//
// tui
//

func confirm(prompt string) bool {
	fmt.Print(prompt + " ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.ToLower(strings.TrimSpace(scanner.Text())) == "y"
}

func printProgress(stopChan chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond)
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

	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	results := make(chan foundDir, 100)

	stopProgress := make(chan struct{})
	var progressWg sync.WaitGroup
	progressWg.Add(1)
	go printProgress(stopProgress, &progressWg)

	// start walker
	wg.Add(1)
	go walkDir(*startDir, 0, sem, &wg, results)

	// close results when done
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
