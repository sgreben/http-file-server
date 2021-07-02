package targz

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
)

func TarGz(w io.Writer, path string) error {
	basePath := path
	addFile := func(w *tar.Writer, path string, stat os.FileInfo) error {
		if stat.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		header := new(tar.Header)
		path, err = filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		header.Name = path
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		if err := w.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(w, file); err != nil {
			return err
		}
		return w.Flush()
	}
	wGzip := gzip.NewWriter(w)
	wTar := tar.NewWriter(wGzip)
	defer func() {
		if err := wTar.Close(); err != nil {
			log.Println(err)
		}
		if err := wGzip.Close(); err != nil {
			log.Println(err)
		}
	}()
	return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return addFile(wTar, path, info)
	})
}
