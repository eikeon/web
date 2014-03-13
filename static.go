package web

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

var PERM = regexp.MustCompile("[0-9a-f]{8}~")

type staticHandler struct {
	root http.FileSystem
}

func NewStaticHandler(root http.FileSystem) *staticHandler {
	return &staticHandler{root}
}

func (fs *staticHandler) Match(r *http.Request, rm *mux.RouteMatch) bool {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		//r.URL.Path = upath
	}
	f, err := fs.root.Open(path.Clean(upath))
	if err == nil {
		d, err1 := f.Stat()
		if err1 != nil {
			return false
		}
		f.Close()
		if d.IsDir() {
			return false
		}
		return true
	} else {
		return false
	}
}

func (fs *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	f, err := fs.root.Open(path.Clean(upath))
	if err == nil {
		defer f.Close()
		d, err1 := f.Stat()
		if err1 != nil {
			http.Error(w, "could not stat file", http.StatusInternalServerError)
			return
		}
		url := r.URL.Path
		if d.IsDir() {
			if url[len(url)-1] != '/' {
				http.Redirect(w, r, url+"/", http.StatusMovedPermanently)
				return
			}
		} else {
			if url[len(url)-1] == '/' {
				http.Redirect(w, r, url[0:len(url)-1], http.StatusMovedPermanently)
				return
			}
		}
		if d.IsDir() {
			w.WriteHeader(http.StatusNotFound)
		} else {
			if PERM.MatchString(path.Base(url)) {
				ttl := int64(365 * 86400)
				w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", ttl))
			}
			http.ServeContent(w, r, d.Name(), d.ModTime(), f)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
