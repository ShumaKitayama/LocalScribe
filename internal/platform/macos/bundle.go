package macos

import (
	"os"
	"path/filepath"
)

// BundledBinDir は .app 内の Contents/Resources/bin を返す。バンドル外なら false。
func BundledBinDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", false
	}
	return BundledBinDirFromExecutable(exe)
}

// BundledBinDirFromExecutable は MacOS 配下の実行ファイルパスから同梱 bin を解決する。
func BundledBinDirFromExecutable(executablePath string) (string, bool) {
	macosDir := filepath.Dir(executablePath)
	if filepath.Base(macosDir) != "MacOS" {
		return "", false
	}
	contentsDir := filepath.Dir(macosDir)
	if filepath.Base(contentsDir) != "Contents" {
		return "", false
	}
	return filepath.Join(contentsDir, "Resources", "bin"), true
}

// BundledToolPath は同梱 bin 内の実行ファイルがあればそのパスを返す。
func BundledToolPath(name string) (string, bool) {
	dir, ok := BundledBinDir()
	if !ok {
		return "", false
	}
	return BundledToolPathInDir(dir, name)
}

// BundledToolPathInDir は dir 内の name が実行可能ならそのパスを返す。
func BundledToolPathInDir(dir, name string) (string, bool) {
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	if info.Mode()&0o111 == 0 {
		return "", false
	}
	return path, true
}
