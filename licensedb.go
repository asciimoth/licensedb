package licensedb

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
)

//go:generate go run genembed.go -url=https://github.com/spdx/license-list-data/archive/refs/tags/v3.27.0.zip -name=spdx3.27.0.zip

var ErrNotFound = errors.New("license not found in archive")

type Archive struct {
	files map[string]*zip.File
	names []string
}

// Load loads embedded Archive.
func Load() *Archive {
	r := bytes.NewReader(archive)
	zr, err := zip.NewReader(r, int64(len(archive)))
	if err != nil {
		// There should not be errors while whorking with embedded archive
		panic(err)
	}

	files := make(map[string]*zip.File, len(zr.File))
	names := make([]string, 0, len(zr.File))

	for _, f := range zr.File {
		files[f.Name] = f
		names = append(names, f.Name)
	}

	return &Archive{
		files: files,
		names: names,
	}
}

// List returns the list of avalable licenses names
func (a *Archive) List() []string {
	out := make([]string, len(a.names))
	copy(out, a.names)
	return out
}

// Get returns the contents of given license.
// If the file isn't found, [ErrNotFound] is returned.
func (a *Archive) Get(name string) (string, error) {
	f, ok := a.files[name]
	if !ok {
		return "", ErrNotFound
	}
	rc, err := f.Open()
	if err != nil {
		// There should not be errors while whorking with embedded archive
		panic(err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		// There should not be errors while whorking with embedded archive
		panic(err)
	}
	return string(data), nil
}
