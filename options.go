package mdoc

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

// Option describes a functional option for configuring the handler.
type Option func(*handler)

// Root sets the URL path to which the handler is mounted at.
// Defaults to the root URL path.
func Root(root string) Option {
	return func(h *handler) {
		h.root = root
	}
}

// defaultRoot is the default URL path.
const defaultRoot = "/"

// Theme sets the path to the templates and assets used to
// render Markdown documents.
func Theme(dir string) Option {
	return func(h *handler) {
		h.themeDir = dir
	}
}

// defaultThemeDir is the relative path to the default theme.
const defaultThemeDir = "contrib/themes/default"

// IndexRenderer sets the IndexPage rendering function.
// Defaults to a basic rendering function.
func IndexRenderer(fn func(v IndexPage) ([]byte, error)) Option {
	return func(h *handler) {
		h.indexRenderer = fn
	}
}

// defaultIndexRenderer returns a default IndexPage renderer.
func defaultIndexRenderer(themeDir string) func(IndexPage) ([]byte, error) {
	t := template.Must(template.ParseFiles(filepath.Join(themeDir, "layout.html")))
	t = template.Must(t.ParseFiles(filepath.Join(themeDir, "index.html")))
	return func(v IndexPage) ([]byte, error) {
		var buf bytes.Buffer
		err := t.Execute(&buf, v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

// DocumentRenderer sets the DocumentPage rendering function.
// Defaults to a basic rendering function.
func DocumentRenderer(fn func(v DocumentPage) ([]byte, error)) Option {
	return func(h *handler) {
		h.documentRenderer = fn
	}
}

// defaultDocumentRenderer returns a the default DocumentPage renderer.
func defaultDocumentRenderer(themeDir string) func(DocumentPage) ([]byte, error) {
	t := template.Must(template.ParseFiles(filepath.Join(themeDir, "layout.html")))
	t = template.Must(t.ParseFiles(filepath.Join(themeDir, "doc.html")))
	return func(v DocumentPage) ([]byte, error) {
		var buf bytes.Buffer
		err := t.Execute(&buf, v)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

// ErrorHandler sets the http.Handler to delegate to when errors are returned.
// Defaults to writing a response with HTTP 404 Not Found if the package fails
// to import, otherwise HTTP 500 Internal Server Error to the response.
func ErrorHandler(fn func(http.ResponseWriter, *http.Request, error)) Option {
	return func(h *handler) {
		h.errorHandler = fn
	}
}

// defaultErrorHandler responds to the request with a plain text error message.
func defaultErrorHandler(w http.ResponseWriter, req *http.Request, err error) {
	var code int
	switch {
	case err == ErrNotFound:
		code = http.StatusNotFound
	case os.IsNotExist(err):
		code = http.StatusNotFound
	case os.IsPermission(err):
		code = http.StatusForbidden
	default:
		code = http.StatusInternalServerError
	}
	http.Error(w, http.StatusText(code), code)
}
