package migrations

import (
	"io"
	"io/ioutil"
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

// CreateDirectory creates the migrations directory on disk if it doesn't
// already exist.
//func (dw *DiskReadWrite) CreateDirectory(directory string) error {
//	return os.MkdirAll(directory, 0755)
//}

// WriteMigration writes the migration file to disk.  Expects a path to the
// migration file.
//func (dw *DiskReadWrite) WriteMigration(path string, migration []byte) error {
//	if err := ioutil.WriteFile(path, migration, 0644); err != nil {
//		return err
//	}
//	return nil
//}

// Files reads the filenames from disk.
func (d *DiskReader) Files(directory string) ([]string, error) {
	files, err := ioutil.ReadDir(directory)
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
