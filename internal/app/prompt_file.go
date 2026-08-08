package app

import (
	"fmt"
	"os"
	"path/filepath"
)

// writeDocumentReviewPrompt never writes through an existing directory entry.
// In force mode it writes a sibling temporary file and renames it into place;
// rename replaces a symlink itself rather than following it.
func writeDocumentReviewPrompt(path string, data []byte, force bool) error {
	if !force {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			return err
		}
		return file.Close()
	}

	temporary, err := os.CreateTemp(filepath.Dir(path), ".default.md.tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace prompt: %w", err)
	}
	return nil
}
