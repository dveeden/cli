package main

import (
	"path"

	rn "github.com/confluentinc/cli/internal/pkg/release-notes"
)

var (
	releaseVersion = "v0.0.0"
	prevVersion    = "v0.0.0"
)

func main() {
	filename := path.Join(".", "release-notes", "prep")
	err := rn.WriteReleaseNotesPrep(filename, releaseVersion, prevVersion)
	if err != nil {
		panic(err)
	}
}
