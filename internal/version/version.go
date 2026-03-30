// Package version provides the single source of truth for the Aegis framework version.
//
// At build time, GoReleaser injects the git tag via:
//
//	-X github.com/theinventorylib/aegis/internal/version.Version=v1.2.3
//
// At runtime (when built without GoReleaser, e.g. library consumers), the version
// is resolved from the embedded module build info via runtime/debug.ReadBuildInfo().
// This covers cases like:
//
//	go get github.com/theinventorylib/aegis@v1.2.3  // version embedded in consumer binary
//	go run ./...                                      // returns "dev"
package version

import "runtime/debug"

// Version is the Aegis framework version.
// GoReleaser overrides this var at link time for release builds.
// Falls back to runtime build info, then "dev" for local/test builds.
var Version = "dev"

func init() {
	if Version != "dev" {
		// GoReleaser already injected the real version.
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	// When aegis itself is the main module (e.g. building the CLI from source
	// without goreleaser), Main.Version holds the module version.
	if info.Main.Path == "github.com/theinventorylib/aegis" &&
		info.Main.Version != "" &&
		info.Main.Version != "(devel)" {
		Version = info.Main.Version
		return
	}

	// When aegis is imported as a library dependency inside another binary,
	// the version appears in the dependency list.
	for _, dep := range info.Deps {
		if dep.Path == "github.com/theinventorylib/aegis" {
			if dep.Version != "" && dep.Version != "(devel)" {
				Version = dep.Version
			}
			return
		}
	}
}
