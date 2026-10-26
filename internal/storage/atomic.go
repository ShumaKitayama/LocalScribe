package storage

import (
	"fmt"
	"os"
)

// renameFile は os.Rename の差し替え用。テストで失敗を注入する。
var renameFile = os.Rename

// WriteFileAtomic は path.tmp に書き、成功したら rename する。失敗時は .tmp を残さない。
func WriteFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write temp file %q: %w", tmp, err)
	}
	if err := renameFile(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %q to %q: %w", tmp, path, err)
	}
	return nil
}
