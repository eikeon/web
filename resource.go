package web

import (
	"log"
	"net/http"
	"net/url"
)

type Resource struct {
	URL         string `db:"HASH"`
	Title       string
	Description string
	Photo       string
	Name        string
	Type        string // open graph type
	Static      string
	Up          string
	Canonical   string
	Redirect    string
	Digest      string
	ContentType string
	Size        int64
}

func (r *Resource) Site() *Resource {
	if u, err := url.Parse(r.URL); err == nil {
		u = u.ResolveReference(&url.URL{Path: "/"})
		if dr, err := Get(u.String()); err == nil {
			return dr
		} else {
			return &Resource{URL: u.String(), Title: "Site Not Found"}
		}
	} else {
		return nil
	}
}

func (r *Resource) Host() string {
	if u, err := url.Parse(r.URL); err == nil {
		return u.Host
	} else {
		log.Println("URL Parse:", err)
		return "parse_error"
	}
}

func (r *Resource) Path() string {
	if u, err := url.Parse(r.URL); err == nil {
		return u.Path
	} else {
		log.Println("URL Parse:", err)
		return "parse_error"
	}
}

type ResourceHandler struct {
	Static  bool
	Root    http.Dir
	Aliases map[string]string
	GetData func(r *Resource) TemplateData `json:"-"`
}

func (rh *ResourceHandler) URL(r *http.Request) *url.URL {
	host := r.Host
	if alias, ok := rh.Aliases[r.Host]; ok {
		host = alias
	}
	u := &url.URL{Host: host, Path: r.URL.String()}
	return u
}

func (rh *ResourceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if rh.Match(r) {
		rh.ServeStatic(w, r)
		return
	}

	url := rh.URL(r).String()

	if dr, err := Get(url); err == nil {
		if dr.Redirect != "" {
			isHTTPS := r.TLS != nil
			scheme := "http"
			if isHTTPS {
				scheme = "https"
			}
			http.Redirect(w, r, scheme+":"+dr.Redirect, http.StatusSeeOther)
			return
		}
		if dr.Name == "404" {
			site := dr.Site()
			if site.Canonical != "" {
				CanonicalHost(site.Canonical)(w, r)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
		t := rh.getTemplate(dr)
		d := rh.GetData(dr)
		if t != nil {
			writeTemplate(t, d, w)
		} else {
			w.Write([]byte("no template found"))
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func CanonicalHost(canonical string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Host != canonical {
			isHTTPS := req.TLS != nil
			scheme := "http"
			if isHTTPS {
				scheme = "https"
			}
			http.Redirect(w, req, scheme+"://"+canonical+req.URL.Path, http.StatusMovedPermanently)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
	}
}
