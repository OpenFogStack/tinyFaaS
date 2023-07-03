package util

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path"
)

func Unzip(zipPath string, p string) error {

	log.Printf("Unzipping %s to %s", zipPath, p)

	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}

	// extract zip
	for _, f := range archive.File {
		log.Printf("Extracting %s", f.Name)

		if f.FileInfo().IsDir() {
			path := path.Join(p, f.Name)
			log.Printf("Creating directory %s in %s", f.Name, path)

			err = os.MkdirAll(path, 0777)
			if err != nil {
				return err
			}
			continue
		}

		// open file
		rc, err := f.Open()
		if err != nil {
			return err
		}

		// create file
		path := path.Join(p, f.Name)
		// err = os.MkdirAll(path, 0777)
		// if err != nil {
		// return err
		// }

		// write file
		w, err := os.Create(path)
		if err != nil {
			return err
		}

		// copy
		_, err = io.Copy(w, rc)
		if err != nil {
			return err
		}

		log.Printf("Extracted %s to %s", f.Name, path)

		// close
		rc.Close()
		w.Close()
	}

	return nil
}
