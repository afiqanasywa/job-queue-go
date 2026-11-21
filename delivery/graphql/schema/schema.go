// Package schema holds the GraphQL schema. The .graphql files are embedded
// into the binary with go:embed, so the server has no runtime file dependency
// and no code generation step is needed.
package schema

import (
	"bytes"
	"embed"
	"io/fs"
	"sort"
)

//go:embed *.graphql type/*.graphql
var files embed.FS

// String concatenates every embedded .graphql file into one schema string.
func String() string {
	var names []string
	_ = fs.WalkDir(files, ".", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, path)
		}
		return err
	})
	sort.Strings(names)

	buf := bytes.Buffer{}
	for _, name := range names {
		b, err := files.ReadFile(name)
		if err != nil {
			panic(err) // embedded files are always readable
		}
		buf.Write(b)
		if len(b) > 0 && b[len(b)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}
