// Package mdoc provides a http.Handler that renders
// a directory of Markdown documents.
package mdoc

import (
	"bytes"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	gfm "github.com/shurcooL/github_flavored_markdown"
)

// handler represents a http.Handler that renders Markdown documents.
type handler struct {
	dir              string
	root             string
	themeDir         string
	assetsDir        string
	indexRenderer    func(v IndexPage) ([]byte, error)
	documentRenderer func(v DocumentPage) ([]byte, error)
	errorHandler     func(w http.ResponseWriter, req *http.Request, err error)
}

// New returns a http.Handler that renders Markdown documents.
func New(dir string, opts ...Option) http.Handler {
	if dir == "" {
		dir = "."
	}
	h := &handler{
		dir:          dir,
		root:         defaultRoot,
		themeDir:     defaultThemeDir,
		errorHandler: defaultErrorHandler,
	}
	for _, option := range opts {
		option(h)
	}
	h.indexRenderer = defaultIndexRenderer(h.themeDir)
	h.documentRenderer = defaultDocumentRenderer(h.themeDir)
	m := http.NewServeMux()
	m.Handle("/.mdoc/assets/", http.StripPrefix("/.mdoc/assets/", h.assets()))
	m.Handle("/", h)
	return m
}

// ServeHTTP implements the http.Handler interface.
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	name := h.dir
	if url != "/" {
		name += path.Clean(url)
	}
	f, err := os.Open(name)
	if err != nil {
		h.errorHandler(w, req, err)
		return
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		h.errorHandler(w, req, err)
		return
	}
	isIndexPage := false
	if fi.IsDir() {
		if !strings.HasSuffix(url, "/") {
			redirect(w, req, path.Base(url)+"/")
			return
		}
		name += "/index.md"
		ff, err := os.Open(name)
		if err == nil {
			defer ff.Close()
			ffi, err := ff.Stat()
			if err == nil {
				f = ff
				fi = ffi
				isIndexPage = true
			}
		}
	}
	var b []byte
	if fi.IsDir() {
		b, err = h.renderIndex(w, f, url)
		if err != nil {
			h.errorHandler(w, req, err)
		}
	} else if !isIndexPage && strings.HasSuffix(url, "/") {
		redirect(w, req, "../"+path.Base(url))
		return
	} else {
		b, err = h.renderDocument(w, f, url)
		if err != nil {
			h.errorHandler(w, req, err)
		}
	}
	http.ServeContent(w, req, name, fi.ModTime(), bytes.NewReader(b))
}

func (h *handler) assets() http.Handler {
	h.assetsDir = filepath.Join(h.themeDir, "assets")
	return http.FileServer(http.Dir(h.assetsDir))
}

// Layout represents the page data used by both
// IndexPage and DocumentPage.
type Layout struct {
	root  string
	path  string
	theme string
}

// Dir returns the path of the current directory.
func (v Layout) Dir() string {
	return "/" + strings.TrimPrefix(v.path, v.root)
}

// StaticFile returns the path to an asset file.
func (v Layout) StaticFile(name string) string {
	return path.Join(v.root, ".mdoc/assets", name)
}

// IndexPage represents the data used to render a directory listing.
type IndexPage struct {
	Layout
	Files []File
}

func (h *handler) renderIndex(w http.ResponseWriter, f *os.File, path string) ([]byte, error) {
	files, err := getFiles(f)
	if err != nil {
		return nil, err
	}
	v := IndexPage{
		Layout: Layout{
			root: h.root,
			path: path,
		},
		Files: files,
	}
	return h.indexRenderer(v)
}

// DocumentPage represents the data used to render a document.
type DocumentPage struct {
	Layout
	Name    string
	Content template.HTML
}

// ErrNotFound represents that the file does not exist or
// is not a Markdown file.
var ErrNotFound = errors.New("mdoc: file not found")

func (h *handler) renderDocument(w http.ResponseWriter, f *os.File, path string) ([]byte, error) {
	name := f.Name()
	if !isMarkdownFile(name) {
		return nil, ErrNotFound
	}
	raw, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	v := DocumentPage{
		Layout: Layout{
			root: h.root,
			path: path,
		},
		Name:    name,
		Content: template.HTML(string(gfm.Markdown(raw))),
	}
	return h.documentRenderer(v)
}

func redirect(w http.ResponseWriter, req *http.Request, loc string) {
	q := req.URL.RawQuery
	if q != "" {
		loc += "?" + q
	}
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func isMarkdownFile(name string) bool {
	ext := filepath.Ext(name)
	for _, e := range []string{".md", ".markdown"} {
		if ext == e {
			return true
		}
	}
	return false
}

func getFiles(f *os.File) ([]File, error) {
	fis, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	var files []File
	for _, fi := range fis {
		f := File{Name: fi.Name(), IsDir: fi.IsDir()}
		if strings.HasPrefix(f.Name, ".") || (!f.IsDir && !isMarkdownFile(f.Name)) {
			continue
		}
		files = append(files, f)
	}
	sort.Sort(byName(files))
	return files, nil
}

// File represents a file for use in a HTML view.
type File struct {
	Name  string
	IsDir bool
}

// DisplayName returns the file name with a forward
// slash appended for directories.
func (f File) DisplayName() string {
	if f.IsDir {
		return f.Name + "/"
	}
	return f.Name
}

type byName []File

func (v byName) Len() int      { return len(v) }
func (v byName) Swap(i, j int) { v[i], v[j] = v[j], v[i] }

func (v byName) Less(i, j int) bool {
	switch {
	case v[i].IsDir && !v[j].IsDir:
		return true
	case !v[i].IsDir && v[j].IsDir:
		return false
	default:
		return v[i].Name < v[j].Name
	}
}
