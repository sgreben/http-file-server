package routes

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Routes struct {
	Separator string

	Values []struct {
		Route string
		Path  string
	}
	Texts []string
}

func (fv *Routes) Help() string {
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}
	return fmt.Sprintf("a route definition ROUTE%sPATH (ROUTE defaults to basename of PATH if omitted)", separator)
}

// Set is flag.Value.Set
func (fv *Routes) Set(v string) error {
	separator := "="
	if fv.Separator != "" {
		separator = fv.Separator
	}
	i := strings.Index(v, separator)
	var route, path string
	var err error
	if i <= 0 {
		path = strings.TrimPrefix(v, "=")
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		route = fmt.Sprintf("/%s/", filepath.Base(path))
	} else {
		route = v[:i]
		path = v[i+len(separator):]
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		if !strings.HasSuffix(route, "/") {
			route = route + "/"
		}
	}
	fv.Texts = append(fv.Texts, v)
	fv.Values = append(fv.Values, struct {
		Route string
		Path  string
	}{
		Route: route,
		Path:  path,
	})
	return nil
}

func (fv *Routes) String() string {
	return strings.Join(fv.Texts, ", ")
}
