package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
)

var Root *string

type Site interface {
	Host() string
	Routes() []*Route
	GetResource(name string, route *Route, vars Vars) Resource
}

type Resource interface{}

type Vars map[string]string

type Route struct {
	Path string
	Name string
	Data map[string]string
	site Site
}

func (r *Route) Site() Site {
	return r.site
}

func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := getTemplate(r.Name)
	res := r.site.GetResource(r.Name, r, mux.Vars(req))
	if res == nil {
		w.WriteHeader(http.StatusNotFound)
	}
	writeTemplate(t, res, w)
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

func handler(j io.Reader, site Site) (http.Handler, error) {
	dec := json.NewDecoder(j)
	if err := dec.Decode(site); err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	router.StrictSlash(true)

	host := site.Host()
	if strings.HasPrefix(host, "www.") {
		router.Host(host[4:len(host)]).HandlerFunc(CanonicalHost(host))
	} else {
		router.Host("www." + host).HandlerFunc(CanonicalHost(host))
	}

	s := router.Host(host).Subrouter()

	for _, r := range site.Routes() {
		r.site = site
		s.Handle(r.Path, r).Name(r.Name).Methods("GET")
	}

	static := NewStaticHandler(http.Dir(path.Join(*Root, "static/")))
	s.MatcherFunc(static.Match).Handler(static)
	s.NewRoute().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		t := getTemplate("404")
		d := map[string]interface{}{"Path": r.URL.Path, "Site": site}
		writeTemplate(t, d, w)
	}))

	router.NewRoute().MatcherFunc(static.Match).Handler(static)

	return router, nil
}

func Handler(site Site) (http.Handler, error) {
	if j, err := os.OpenFile(path.Join(*Root, "pages.json"), os.O_RDONLY, 0666); err == nil {
		s, err := handler(j, site)
		j.Close()
		if err == nil {
			return s, nil
		} else {
			return nil, errors.New("WARNING: could not decode pages.json: " + err.Error())
		}
	} else {
		return nil, errors.New("WARNING: could not open pages.json: " + err.Error())
	}
}
