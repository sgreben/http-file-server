package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
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

const directoryListingTemplateText = `
<html>
<head>
	<title>{{ .Title }}</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>body{font-family: sans-serif;}td{padding:.5em;}a{display:block;}tbody tr:nth-child(odd){background:#eee;}.number{text-align:right}.text{text-align:left;word-break:break-all;}canvas,table{width:100%;max-width:100%;}</style>
</head>
<body>
<h1>{{ .Title }}</h1>
{{ if .Files }}
<table>
	<thead>
		<th></th>
		<th colspan=2 class=number>Size (bytes)</th>
	</thead>
	<tbody>
	<tr><td colspan=3><a href="{{ .TarGzURL }}">.tar.gz of all files</a></td></tr>
	<tr><td colspan=3><a href="{{ .ZipURL }}">.zip of all files</a></td></tr>
	{{- range .Files }}
	<tr>
		{{ if (not .IsDir) }}
		<td class=text><a href="{{ .URL.String }}">{{ .Name }}</td>
		<td class=number>{{.Size.String }}</td>
		<td class=number>({{ .Size | printf "%d" }})</td>
		{{ else }}
		<td colspan=3 class=text><a href="{{ .URL.String }}">{{ .Name }}</td>
		{{ end }}
	</tr>
	{{end -}}
	</tbody>
</table>
{{ end }}
</body>
</html>
`

type fileSizeBytes int64

func (f fileSizeBytes) String() string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	switch {
	case f < KB:
		return fmt.Sprintf("%d", f)
	case f < MB:
		return fmt.Sprintf("%dK", f/KB)
	case f < GB:
		return fmt.Sprintf("%dM", f/MB)
	case f >= GB:
		fallthrough
	default:
		return fmt.Sprintf("%dG", f/GB)
	}
}

type directoryListingFileData struct {
	Name  string
	Size  fileSizeBytes
	IsDir bool
	URL   *url.URL
}

type directoryListingData struct {
	Title    string
	ZipURL   *url.URL
	TarGzURL *url.URL
	Files    []directoryListingFileData
}

type fileHandler struct {
	route string
	path  string
}

var (
	directoryListingTemplate = template.Must(template.New("").Parse(directoryListingTemplateText))
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
	directoryListingTemplate.Execute(w, directoryListingData{
		Title: func() string {
			relPath, _ := filepath.Rel(f.path, dirPath)
			return filepath.Join(filepath.Base(f.path), relPath)
		}(),
		TarGzURL: func() *url.URL {
			url := *r.URL
			q := url.Query()
			q.Set(tarGzKey, tarGzValue)
			url.RawQuery = q.Encode()
			return &url
		}(),
		ZipURL: func() *url.URL {
			url := *r.URL
			q := url.Query()
			q.Set(zipKey, zipValue)
			url.RawQuery = q.Encode()
			return &url
		}(),
		Files: func() (out []directoryListingFileData) {
			for _, d := range files {
				name := d.Name()
				if d.IsDir() {
					name += "/"
				}
				fileData := directoryListingFileData{
					Name:  name,
					IsDir: d.IsDir(),
					Size:  fileSizeBytes(d.Size()),
					URL: func() *url.URL {
						url := *r.URL
						path := filepath.Join(url.Path, name)
						if d.IsDir() {
							path += "/"
						}
						url.Path = path
						return &url
					}(),
				}
				out = append(out, fileData)
			}
			return out
		}(),
	})
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
