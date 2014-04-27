package web

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

var PERM = regexp.MustCompile("[0-9a-f]{8}~")

func (rh *ResourceHandler) Match(r *http.Request) bool {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		//r.URL.Path = upath
	}
	u := rh.URL(r)
	upath = "/" + u.Host + "/static" + upath
	f, err := rh.Root.Open(path.Clean(upath))
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

func (rh *ResourceHandler) ServeStatic(w http.ResponseWriter, r *http.Request) {
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	u := rh.URL(r)
	upath = "/" + u.Host + "/static" + upath
	f, err := rh.Root.Open(path.Clean(upath))
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

func paths(root string) <-chan string {
	ch := make(chan string)
	go func() {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() == false {
				if p, err2 := filepath.Rel(root, path); err2 == nil {
					ch <- p
				} else {
					log.Println(err2)
				}
			}
			return err
		}); err != nil {
			log.Println("walk:", err)
		}
		close(ch)
	}()

	return ch

}

var digests map[string]map[string]string

func (rh *ResourceHandler) Init(bucket string, host string) {
	if digests == nil {
		digests = make(map[string]map[string]string)
	}
	if digests[host] == nil {
		digests[host] = make(map[string]string)
	}
	auth, err := aws.EnvAuth()
	if err != nil {
		log.Fatal(err)
	}

	s := s3.New(auth, aws.USEast)

	b := s.Bucket(bucket)
	err = b.PutBucket(s3.PublicRead)
	if err != nil {
		log.Fatal(err)
	}

	sroot := path.Join(string(rh.Root), host, "static")
	for p := range paths(sroot) {
		fp := path.Join(sroot, p)

		fi, err := os.Stat(fp)
		if err != nil {
			log.Println("stat err:", err)
		}

		d := sha1.New()
		f, err := os.Open(fp)
		if err != nil {
			log.Println("open err:", err)
		}
		digest := ""
		if b, err := ioutil.ReadAll(f); err == nil {
			if _, werr := d.Write(b); werr == nil {
				digest = fmt.Sprintf("sha1-%x", d.Sum(nil))
			} else {
				log.Println("write err:", werr)
			}
		} else {
			log.Println("read all err:", err)
		}

		pp := path.Join("/", p)
		digests[host][pp] = digest

		if r, err := b.GetReader(digest); err == nil {
			r.Close()
		} else {
			f.Seek(0, 0)
			reader := bufio.NewReader(f)
			length := fi.Size()
			ctype := mime.TypeByExtension(filepath.Ext(p))
			err = b.PutReader(digest, reader, length, ctype, s3.PublicRead)
			if err != nil {
				log.Println("ERR:", err)
			}
			log.Println(p, digest, fi.Size())
			u := url.URL{Host: host, Path: p}
			r := &Resource{URL: u.String(), Digest: digest, ContentType: ctype, Size: fi.Size()}
			Put(r)
		}
		f.Close()
	}
}
