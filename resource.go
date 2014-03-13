package web

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gorilla/mux"
)

var Root *string

type Resource interface{}

type Vars map[string]string

type Getters map[string]Getter
type Getter func(route *Route, vars Vars) Resource

type Route struct {
	Path   string
	Name   string
	Data   map[string]string
	getter Getter
}

func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := getTemplate(r.Name)
	res := r.getter(r, mux.Vars(req))
	if res == nil {
		w.WriteHeader(http.StatusNotFound)
	}
	writeTemplate(t, res, w)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	t := getTemplate("404")
	writeTemplate(t, nil, w)
}

func NotFoundHandler() http.Handler {
	return http.HandlerFunc(NotFound)
}

func CanonicalHost(canonical string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Host != canonical {
			http.Redirect(w, req, req.URL.Scheme+"://"+canonical+req.URL.Path, http.StatusMovedPermanently)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
	}
}

func handler(j io.Reader, host string, getters map[string]Getter) (http.Handler, error) {
	dec := json.NewDecoder(j)
	var routes []*Route
	if err := dec.Decode(&routes); err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	router.StrictSlash(true)

	if strings.HasPrefix(host, "www.") {
		router.Host(host[4:len(host)]).HandlerFunc(CanonicalHost(host))
	}

	s := router.Host(host).Subrouter()

	for _, r := range routes {
		if g := getters[r.Name]; g == nil {
			log.Printf("Warning: no web.Getter for '%s'\n", r.Name)
		} else {
			r.getter = g
			s.Handle(r.Path, r).Name(r.Name).Methods("GET")
		}
	}

	static := NewStaticHandler(http.Dir(path.Join(*Root, "static/")))
	s.MatcherFunc(static.Match).Handler(static)
	s.NewRoute().Handler(NotFoundHandler())

	return router, nil
}

func Handler(host string, getters map[string]Getter) (http.Handler, error) {
	if j, err := os.OpenFile(path.Join(*Root, "pages.json"), os.O_RDONLY, 0666); err == nil {
		s, err := handler(j, host, getters)
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
