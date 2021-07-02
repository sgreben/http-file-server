package httpfileserver

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sgreben/httpfileserver/internal/targz"
	"github.com/sgreben/httpfileserver/internal/zip"
)

const (
	tarGzKey         = "tar.gz"
	tarGzValue       = "true"
	tarGzContentType = "application/x-tar+gzip"

	zipKey         = "zip"
	zipValue       = "true"
	zipContentType = "application/zip"

	osPathSeparator = string(filepath.Separator)
)

const directoryListingTemplateText = `
<html>
<meta name="google" content="notranslate"/>
<head>
	<title>{{ .Title }}</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<style>body{font-family: sans-serif;}td{padding:.5em;}a{display:block;}tbody tr:nth-child(odd){background:#eee;}.number{text-align:right}.text{text-align:left;word-break:break-all;}canvas,table{width:100%;max-width:100%;}</style>
</head>
<body>
<h1>{{ .Title }}</h1>
{{ if or .Files .AllowUpload }}
<table>
	<thead>
		<th></th>
		<th colspan=2 class=number>Size (bytes)</th>
	</thead>
	<tbody>
	{{- if .Files }}
	<tr><td colspan=3><a href="{{ .TarGzURL }}">.tar.gz of all files</a></td></tr>
	<tr><td colspan=3><a href="{{ .ZipURL }}">.zip of all files</a></td></tr>
	{{- end }}
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
	{{- end }}
	{{- if .AllowUpload }}
	<tr><td colspan=3><form method="post" enctype="multipart/form-data"><input required name="file" type="file"/><input value="Upload" type="submit"/></form></td></tr>
	{{- end }}
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
	divBy := func(x int64) int {
		return int(math.Round(float64(f) / float64(x)))
	}
	switch {
	case f < KB:
		return fmt.Sprintf("%d", f)
	case f < MB:
		return fmt.Sprintf("%dK", divBy(KB))
	case f < GB:
		return fmt.Sprintf("%dM", divBy(MB))
	case f >= GB:
		fallthrough
	default:
		return fmt.Sprintf("%dG", divBy(GB))
	}
}

type directoryListingFileData struct {
	Name  string
	Size  fileSizeBytes
	IsDir bool
	URL   *url.URL
}

type directoryListingData struct {
	Title       string
	ZipURL      *url.URL
	TarGzURL    *url.URL
	Files       []directoryListingFileData
	AllowUpload bool
}

type fileHandler struct {
	route       string
	path        string
	allowUpload bool
}

var (
	directoryListingTemplate = template.Must(template.New("").Parse(directoryListingTemplateText))
)

func (f *fileHandler) serveStatus(w http.ResponseWriter, r *http.Request, status int) error {
	w.WriteHeader(status)
	_, err := w.Write([]byte(http.StatusText(status)))
	if err != nil {
		return err
	}
	return nil
}

func (f *fileHandler) serveTarGz(w http.ResponseWriter, r *http.Request, path string) error {
	w.Header().Set("Content-Type", tarGzContentType)
	name := filepath.Base(path) + ".tar.gz"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return targz.TarGz(w, path)
}

func (f *fileHandler) serveZip(w http.ResponseWriter, r *http.Request, osPath string) error {
	w.Header().Set("Content-Type", zipContentType)
	name := filepath.Base(osPath) + ".zip"
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, name))
	return zip.Zip(w, osPath)
}

func (f *fileHandler) serveDir(w http.ResponseWriter, r *http.Request, osPath string) error {
	d, err := os.Open(osPath)
	if err != nil {
		return err
	}
	files, err := d.Readdir(-1)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return directoryListingTemplate.Execute(w, directoryListingData{
		AllowUpload: f.allowUpload,
		Title: func() string {
			relPath, _ := filepath.Rel(f.path, osPath)
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
					name += osPathSeparator
				}
				fileData := directoryListingFileData{
					Name:  name,
					IsDir: d.IsDir(),
					Size:  fileSizeBytes(d.Size()),
					URL: func() *url.URL {
						url := *r.URL
						url.Path = path.Join(url.Path, name)
						if d.IsDir() {
							url.Path += "/"
						}
						return &url
					}(),
				}
				out = append(out, fileData)
			}
			return out
		}(),
	})
}

func (f *fileHandler) serveUploadTo(w http.ResponseWriter, r *http.Request, osPath string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	in, h, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		w.Header().Set("Location", r.URL.String())
		w.WriteHeader(http.StatusSeeOther)
	}
	if err != nil {
		return err
	}
	outPath := filepath.Join(osPath, filepath.Base(h.Filename))
	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	w.Header().Set("Location", r.URL.String())
	w.WriteHeader(http.StatusSeeOther)
	return nil
}

// ServeHTTP is http.Handler.ServeHTTP
func (f *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] %s %s %s", f.path, r.RemoteAddr, r.Method, r.URL.String())
	urlPath := r.URL.Path
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	urlPath = strings.TrimPrefix(urlPath, f.route)
	urlPath = strings.TrimPrefix(urlPath, "/"+f.route)

	osPath := strings.ReplaceAll(urlPath, "/", osPathSeparator)
	osPath = filepath.Clean(osPath)
	osPath = filepath.Join(f.path, osPath)
	info, err := os.Stat(osPath)
	switch {
	case os.IsNotExist(err):
		_ = f.serveStatus(w, r, http.StatusNotFound)
	case os.IsPermission(err):
		_ = f.serveStatus(w, r, http.StatusForbidden)
	case err != nil:
		_ = f.serveStatus(w, r, http.StatusInternalServerError)
	case r.URL.Query().Get(zipKey) != "":
		err := f.serveZip(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case r.URL.Query().Get(tarGzKey) != "":
		err := f.serveTarGz(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case f.allowUpload && info.IsDir() && r.Method == http.MethodPost:
		err := f.serveUploadTo(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	case info.IsDir():
		err := f.serveDir(w, r, osPath)
		if err != nil {
			_ = f.serveStatus(w, r, http.StatusInternalServerError)
		}
	default:
		http.ServeFile(w, r, osPath)
	}
}
