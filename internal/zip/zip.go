package zip

import (
	zipper "archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Zip(w io.Writer, path string) error {
	basePath := path
	addFile := func(w *zipper.Writer, path string, stat os.FileInfo) error {
		if stat.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		path, err = filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		zw, err := w.Create(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(zw, file); err != nil {
			return err
		}
		return w.Flush()
	}
	wZip := zipper.NewWriter(w)
	defer func() {
		if err := wZip.Close(); err != nil {
			log.Println(err)
		}
	}()
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return addFile(wZip, path, info)
	})
}
