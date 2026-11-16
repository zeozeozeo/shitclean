package main

import (
	"runtime"
	"strings"
)

var commonSkip = map[string]struct{}{
	".git": {}, ".hg": {}, ".svn": {},

	"node_modules": {}, "target": {}, "build": {}, "bin": {}, "obj": {}, "dist": {},
	"CMakeFiles": {}, "buck-out": {}, "bazel-out": {}, "bazel-bin": {}, "bazel-testlogs": {},

	".gradle": {}, ".dub": {}, ".build": {}, ".terraform": {}, ".next": {}, ".nuxt": {},
	".cache": {}, ".local": {}, ".config": {},
	".cargo": {}, ".rustup": {}, ".npm": {}, ".nuget": {}, ".m2": {}, ".ivy2": {}, ".sbt": {},
	".stack": {}, ".ccls-cache": {}, ".direnv": {},

	"__pycache__": {}, ".venv": {}, "venv": {}, ".tox": {}, ".nox": {},
	".pytest_cache": {}, ".mypy_cache": {}, ".ruff_cache": {}, ".ipo": {},

	".idea": {}, ".vscode": {}, "lost+found": {},
}

var darwinSkip = map[string]struct{}{
	"Library": {}, ".Trashes": {}, ".Spotlight-V100": {}, ".fseventsd": {},
	".DocumentRevisions-V100": {}, ".TemporaryItems": {},
}

var windowsSkip = map[string]struct{}{
	"AppData": {}, "Windows": {}, "Program Files": {}, "Program Files (x86)": {},
	"ProgramData": {}, "WinSxS": {}, "$RECYCLE.BIN": {}, "System Volume Information": {},
	"$WinREAgent": {}, "Recovery": {},
}

var linuxSkip = map[string]struct{}{
	"snap": {},
}

func defaultSkip(name string) bool {
	if strings.HasPrefix(name, ".Trash-") || strings.HasPrefix(name, "bazel-") {
		return true
	}
	if _, ok := commonSkip[name]; ok {
		return true
	}
	switch runtime.GOOS {
	case "darwin":
		_, ok := darwinSkip[name]
		return ok
	case "windows":
		_, ok := windowsSkip[name]
		return ok
	default:
		_, ok := linuxSkip[name]
		return ok
	}
}
