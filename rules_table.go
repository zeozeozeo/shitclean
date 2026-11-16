package main

import "strings"

var ruleTable = []*rule{
	{name: "cmake", files: []string{"CMakeCache.txt"}, dirs: []string{"CMakeFiles"}, targets: []string{""}},
	{name: "cargo", files: []string{"Cargo.toml"}, targets: []string{"target"}},
	{name: "maven", files: []string{"pom.xml"}, targets: []string{"target"}},
	{name: "gradle", files: []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"}, targets: []string{"build", ".gradle", ".kotlin"}},
	{name: "sbt", files: []string{"build.sbt"}, targets: []string{"target", ".bloop", ".metals", ".bsp"}},
	{name: "bazel", files: []string{"WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel", "BUILD", "BUILD.bazel"}, custom: bazelTargets},
	{name: "meson", files: []string{"meson.build"}, targets: []string{"build", "_build", "meson-private"}},
	{name: "ninja", files: []string{"build.ninja"}, targets: []string{"build"}},
	{name: "autotools", files: []string{"configure.ac", "configure.in", "Makefile.am"}, targets: []string{"autom4te.cache", ".deps"}},
	{name: "dune", files: []string{"dune-project", "dune-workspace"}, suffixes: []string{".opam"}, targets: []string{"_build"}},
	{name: "cabal", suffixes: []string{".cabal"}, targets: []string{"dist-newstyle", "dist"}},
	{name: "stack", files: []string{"stack.yaml", "stack.yaml.lock"}, targets: []string{".stack-work"}},
	{name: "buck", targets: []string{"buck-out"}},
	{name: "pants", targets: []string{".pants.d"}},

	{name: "node", targets: []string{"node_modules"}},
	{name: "pnpm", files: []string{"pnpm-lock.yaml"}, targets: []string{"node_modules", ".pnpm-store"}},
	{name: "bun", files: []string{"bun.lockb", "bun.lock"}, targets: []string{"node_modules", ".bun"}},
	{name: "yarn", files: []string{"yarn.lock"}, targets: []string{".yarn/cache", ".yarn/unplugged"}},
	{name: "expo", files: []string{"app.json", "app.config.js", "app.config.ts"}, targets: []string{".expo", ".expo-shared"}},
	{name: "gatsby", files: []string{"gatsby-config.js", "gatsby-config.ts"}, targets: []string{".cache", "public"}},
	{name: "storybook", dirs: []string{".storybook"}, targets: []string{"storybook-static"}},
	{name: "angular", files: []string{"angular.json"}, targets: []string{"dist", ".angular"}},
	{name: "docusaurus", files: []string{"docusaurus.config.js", "docusaurus.config.ts"}, targets: []string{".docusaurus", "build"}},
	{name: "js-cache", targets: []string{".next", ".nuxt", ".output", ".svelte-kit", ".astro", ".turbo", ".parcel-cache"}},

	{name: "python", files: []string{"setup.py", "setup.cfg", "pyproject.toml", "requirements.txt", "Pipfile", "poetry.lock", "uv.lock"}, targets: []string{"build", "dist", ".venv", "venv", ".tox", ".nox", "htmlcov"}},
	{name: "python-egg", dirSuffixes: []string{".egg-info"}},
	{name: "python-cache", targets: []string{"__pycache__", ".pytest_cache", ".mypy_cache", ".ruff_cache", ".pytype", ".hypothesis", ".ipynb_checkpoints"}},

	{name: "zig", files: []string{"build.zig", "build.zig.zon"}, targets: []string{".zig-cache", "zig-cache", "zig-out"}},
	{name: "nim", suffixes: []string{".nimble"}, targets: []string{"nimcache"}},
	{name: "dub", files: []string{"dub.json", "dub.sdl"}, targets: []string{".dub"}},
	{name: "jai", suffixes: []string{".jai"}, targets: []string{"bin", ".build"}},
	{name: "swift", files: []string{"Package.swift"}, targets: []string{".build"}},
	{name: "xcode", dirSuffixes: []string{".xcodeproj", ".xcworkspace"}, targets: []string{"DerivedData", "build"}},
	{name: "cocoapods", files: []string{"Podfile"}, targets: []string{"Pods"}},
	{name: "carthage", files: []string{"Cartfile"}, targets: []string{"Carthage"}},

	{name: "bundler", files: []string{"Gemfile"}, targets: []string{"vendor/bundle", ".bundle"}},
	{name: "composer", files: []string{"composer.json"}, targets: []string{"vendor"}},
	{name: "perl", files: []string{"Makefile.PL", "Build.PL"}, targets: []string{"blib", "_build"}},

	{name: "flutter", files: []string{"pubspec.yaml"}, targets: []string{"build", ".dart_tool", ".flutter-plugins", ".flutter-plugins-dependencies", "ios/Pods", "android/.gradle"}},

	{name: "mix", files: []string{"mix.exs"}, targets: []string{"_build", "deps", ".elixir_ls"}},
	{name: "rebar", files: []string{"rebar.config"}, targets: []string{"_build", "deps"}},

	{name: "dotnet", suffixes: []string{".csproj", ".fsproj", ".vbproj", ".sln", ".vcxproj"}, targets: []string{"bin", "obj", "TestResults"}},
	{name: "unity", dirs: []string{"ProjectSettings"}, targets: []string{"Library", "Temp", "Logs", "obj", "Builds", "UserSettings", "MemoryCaptures"}},
	{name: "unreal", suffixes: []string{".uproject"}, targets: []string{"Intermediate", "Saved", "Binaries", "DerivedDataCache", ".vs"}},
	{name: "android", files: []string{"AndroidManifest.xml"}, targets: []string{"build", ".cxx", ".externalNativeBuild"}},
	{name: "godot", files: []string{"project.godot"}, targets: []string{".godot", ".import"}},

	{name: "clojure", files: []string{"deps.edn", "project.clj", "shadow-cljs.edn"}, targets: []string{".cpcache", "target", ".shadow-cljs"}},
	{name: "elm", files: []string{"elm.json"}, targets: []string{"elm-stuff"}},
	{name: "purescript", files: []string{"spago.yaml", "spago.dhall"}, targets: []string{".spago", "output"}},
	{name: "gleam", files: []string{"gleam.toml"}, targets: []string{"build"}},
	{name: "rescript", files: []string{"rescript.json", "bsconfig.json"}, targets: []string{"lib/bs"}},

	{name: "hugo", files: []string{"hugo.toml", "hugo.yaml", "hugo.json"}, targets: []string{"resources/_gen", "public"}},
	{name: "jekyll", files: []string{"_config.yml"}, targets: []string{"_site", ".jekyll-cache", ".sass-cache"}},
	{name: "mkdocs", files: []string{"mkdocs.yml", "mkdocs.yaml"}, targets: []string{"site"}},

	{name: "terraform", targets: []string{".terraform"}},
	{name: "sam", files: []string{"template.yaml", "template.yml", "samconfig.toml"}, targets: []string{".aws-sam"}},
	{name: "cdk", files: []string{"cdk.json"}, targets: []string{"cdk.out"}},
	{name: "serverless", files: []string{"serverless.yml", "serverless.yaml"}, targets: []string{".serverless"}},

	{name: "ccls", targets: []string{".ccls-cache"}},
	{name: "direnv", targets: []string{".direnv"}},
	{name: "qobs", files: []string{"Qobs.toml"}, targets: []string{"build"}},
}

func init() {
	allRules = ruleTable
	compileRules()
}

func bazelTargets(d *dirInfo) []string {
	var out []string
	for _, name := range d.dirNames {
		if strings.HasPrefix(name, "bazel-") {
			out = append(out, name)
		}
	}
	return out
}
