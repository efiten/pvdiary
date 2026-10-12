package pvdiary

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var embedded embed.FS

// WebFS is de subtree met de gebouwde frontend.
var WebFS fs.FS = mustSub(embedded, "web/dist")

func mustSub(e embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(e, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
