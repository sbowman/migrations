package migrations

import (
	"io"
	"os"
)

// Reader interface allows the migrations to be read from different sources,
// such as an S3 bucket or another data store.
type Reader interface {
	// Files returns the files in a directory or remote path.
	Files(directory string) ([]string, error)

	// Read the SQL from the migration.
	Read(path string) (io.Reader, error)
}

// DiskReader outputs to disk, the Migrations default.
type DiskReader struct {
}

// Files reads the filenames from disk.
func (d *DiskReader) Files(directory string) ([]string, error) {
	files, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, info := range files {
		paths = append(paths, info.Name())
	}

	return paths, nil
}

// Read the SQL migration from disk.
func (d *DiskReader) Read(path string) (io.Reader, error) {
	return os.Open(path)
}
