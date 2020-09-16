package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

type hcard struct {
	Source string `json:"source,omitempty"`
	PName  string `json:"pname,omitempty"`
	Photo  string `json:"uphoto,omitempty"`
}

func copyHeader(m map[string][]string, w http.ResponseWriter, h string) {
	h = http.CanonicalHeaderKey(h)
	if vv, ok := m[h]; ok {
		w.Header().Del(h)
		for _, v := range vv {
			w.Header().Add(h, v)
		}
	}
}

func fetchHcard(link string) (*hcard, *http.Header, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	res, err := http.Get(u.String())
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	i := getRepresentativeHcard(res.Body, res.Request.URL)
	if i == nil {
		return nil, &res.Header, fmt.Errorf("no representative h-card found")
	}

	var hc hcard
	hc.Source = res.Request.URL.String()

	for _, t := range i.Type {
		switch t {
		case "h-card":
			hc.Photo = parseProperty(i, "photo")
			hc.PName = parseProperty(i, "name")
		}
	}

	return &hc, &res.Header, nil
}

func getHcard(link string) (*hcard, map[string][]string) {
	hc, hd, err := fetchHcard(link)
	if err != nil {
		h := hcard{}
		hd := map[string][]string{}
		return &h, hd
	}
	return hc, *hd
}

func getModTime(res *http.Response) time.Time {
	lm, ok := res.Header["Last-Modified"]
	if !ok {
		return time.Now()
	}
	if len(lm) != 1 {
		return time.Now()
	}

	t, err := time.Parse(time.RFC1123, lm[0])
	if err != nil {
		return time.Now()
	}
	return t
}

func setResponseHeaders(w http.ResponseWriter, h map[string][]string) {
	copyHeader(h, w, "cache-control")
	copyHeader(h, w, "expires")

	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveHcard(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Form["url"]) < 1 {
		http.Error(w, "no URL specified", http.StatusBadRequest)
		return
	}
	hc, hd := getHcard(req.Form["url"][0])

	js, err := json.Marshal(hc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	copyHeader(hd, w, "last-modified")
	setResponseHeaders(w, hd)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

func servePhoto(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Form["url"]) < 1 {
		http.Error(w, "no URL specified", http.StatusBadRequest)
		return
	}
	hc, _ := getHcard(req.Form["url"][0])

	if hc.Photo == "" {
		http.Error(w, "no photo", http.StatusNotFound)
		return
	}

	res, err := http.Get(hc.Photo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	bb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := getModTime(res)
	hd := res.Header
	setResponseHeaders(w, hd)
	http.ServeContent(w, req, "", t, bytes.NewReader(bb))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/hcard", serveHcard)
	http.HandleFunc("/api/photo", servePhoto)

	_ = http.ListenAndServe(":"+port, nil)
}
