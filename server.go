package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	tarGzKey         = "tar.gz"
	tarGzValue       = "true"
	tarGzContentType = "application/x-tar+gzip"

	zipKey         = "zip"
	zipValue       = "true"
	zipContentType = "application/zip"
)

type fileHandler struct {
	route string
	path  string
}

var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&#34;",
	"'", "&#39;",
)

func (f *fileHandler) serveStatus(w http.ResponseWriter, r *http.Request, status int) {
	w.WriteHeader(status)
	w.Write([]byte(http.StatusText(status)))
}

func (f *fileHandler) serveTarGz(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Type", tarGzContentType)
	name := filepath.Base(path) + ".tar.gz"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	tarGz(w, path)
}

func (f *fileHandler) serveZip(w http.ResponseWriter, r *http.Request, path string) {
	w.Header().Set("Content-Type", zipContentType)
	name := filepath.Base(path) + ".zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	zip(w, path)
}

func (f *fileHandler) serveDir(w http.ResponseWriter, r *http.Request, dirPath string) {
	d, err := os.Open(dirPath)
	if err != nil {
		f.serveStatus(w, r, http.StatusInternalServerError)
		return
	}
	files, err := d.Readdir(-1)
	if err != nil {
		f.serveStatus(w, r, http.StatusInternalServerError)
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	title, _ := filepath.Rel(f.path, dirPath)
	title = filepath.Join(filepath.Base(f.path), title)
	fmt.Fprintf(w, "<h1>%s</h1>\n", htmlReplacer.Replace(title))
	fmt.Fprintf(w, "<ul>\n")
	for _, d := range files {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		url := url.URL{Path: path.Join(r.URL.Path, name)}
		fmt.Fprintf(w, "<li><a href=\"%s\">%s</a></li>\n", url.String(), htmlReplacer.Replace(name))
	}
	fmt.Fprintf(w, "</ul>\n")
	if len(files) > 0 {
		url := url.URL{Path: r.URL.Path}
		q := url.Query()
		q.Set(tarGzKey, tarGzValue)
		url.RawQuery = q.Encode()
		fmt.Fprintf(w, "<p>\n")
		fmt.Fprintf(w, "<a href=\"%s\">Entire directory as .tar.gz</a>\n", url.String())
		fmt.Fprintf(w, "</p>\n")
		url.RawQuery = ""
		q = url.Query()
		q.Set(zipKey, zipValue)
		url.RawQuery = q.Encode()
		fmt.Fprintf(w, "<p>\n")
		fmt.Fprintf(w, "<a href=\"%s\">Entire directory as .zip</a>\n", url.String())
		fmt.Fprintf(w, "</p>\n")
	}
}

// ServeHTTP is http.Handler.ServeHTTP
func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimPrefix(path, f.route)
	path = strings.TrimPrefix(path, "/"+f.route)
	path = filepath.Clean(path)
	path = filepath.Join(f.path, path)
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		f.serveStatus(w, r, http.StatusNotFound)
	case os.IsPermission(err):
		f.serveStatus(w, r, http.StatusForbidden)
	case err != nil:
		f.serveStatus(w, r, http.StatusInternalServerError)
	case r.URL.Query().Get(zipKey) != "":
		f.serveZip(w, r, path)
	case r.URL.Query().Get(tarGzKey) != "":
		f.serveTarGz(w, r, path)
	case info.IsDir():
		f.serveDir(w, r, path)
	default:
		http.ServeFile(w, r, path)
	}
}
