package web

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

var templates = make(map[string]*template.Template)

func (rh *ResourceHandler) getTemplate(r *Resource) *template.Template {
	page := r.Name
	if page == "" {
		return nil
	}
	site := r.Site()
	name := site.URL + page
	if t, ok := templates[name]; ok {
		return t
	} else {
		if templates[site.URL] == nil {
			sf := func(path string) string {
				return path
			}
			if rh.Static == false {
				sf = func(path string) string {
					d := digests[site.Host()][path]
					log.Println(path, "->", d)
					if d != "" {
						u := url.URL{Host: site.Static, Path: d}
						log.Println("U:", u.String())
						return u.String()
					}
					log.Println("no digest found for:", path)
					return path
				}
			}
			templates[site.URL] = template.Must(template.New("site").Funcs(template.FuncMap{
				"static": sf}).ParseFiles(path.Join(string(rh.Root), site.Host(), "templates/site.html")))
		}
		t, err := templates[site.URL].Clone()

		if err != nil {
			log.Fatal("cloning site: ", err)
		}
		t = template.Must(t.ParseFiles(path.Join(string(rh.Root), site.Host(), "templates/"+page+".html")))

		templates[name] = t
		return t
	}
}

type TemplateData interface{}

func writeTemplate(t *template.Template, d TemplateData, w http.ResponseWriter) {
	var bw bytes.Buffer
	h := md5.New()
	mw := io.MultiWriter(&bw, h)
	err := t.ExecuteTemplate(mw, "html", d)
	if err == nil {
		w.Header().Set("ETag", fmt.Sprintf(`"%x"`, h.Sum(nil)))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", bw.Len()))
		w.Write(bw.Bytes())
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
