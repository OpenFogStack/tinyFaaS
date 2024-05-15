package util

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func CopyFileFromEmbed(src embed.FS, srcPath string, dstPath string) (err error) {
	srcFile, err := src.Open(srcPath)
	if err != nil {
		return
	}
	defer srcFile.Close()

	_, err = os.Stat(dstPath)
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		log.Printf("Destination file %s already exists, skipping", dstPath)
		return
	}

	err = os.MkdirAll(filepath.Dir(dstPath), 0755)
	if err != nil {
		return
	}

	dstFile, err := os.Create(dstPath)

	if err != nil {
		return
	}

	defer func() {
		if e := dstFile.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(dstFile, srcFile)

	if err != nil {
		return
	}

	err = dstFile.Sync()

	if err != nil {
		return
	}

	return

}

func CopyDirFromEmbed(src embed.FS, srcPath string, dstPath string) (err error) {
	entries, err := fs.ReadDir(src, srcPath)

	if err != nil {
		return
	}

	err = os.MkdirAll(dstPath, 0755)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcPath, entry.Name())
		dstPath := filepath.Join(dstPath, entry.Name())

		if entry.IsDir() {
			err = CopyDirFromEmbed(src, srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}

			err = CopyFileFromEmbed(src, srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return

}
