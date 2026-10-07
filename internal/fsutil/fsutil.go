// Package fsutil contains small filesystem helpers.
package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to dir/name via a temp file + rename so readers
// never observe a partially written file. The file is created with 0600
// permissions and dir with 0700.
func WriteFileAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, name+"-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, 0o600)
	return os.Rename(tmpName, filepath.Join(dir, name))
}
