package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"willnorris.com/go/microformats"
)

type hcard struct {
	Source string `json:"source,omitempty"`
	PName  string `json:"pname,omitempty"`
	Photo  string `json:"uphoto,omitempty"`
}

func fetchHcard(link string) (*hcard, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	d := microformats.Parse(res.Body, res.Request.URL)

	var hc hcard
	hc.Source = res.Request.URL.String()

	for _, i := range d.Items {
		for _, t := range i.Type {
			switch t {
			case "h-card":
				hc.Photo = parsePhotoUrl(i)
				hc.PName = parsePName(i)
			}
		}
	}

	return &hc, nil
}

func parsePhotoUrl(mf *microformats.Microformat) (url string) {
	if len(mf.Properties["photo"]) < 1 {
		return
	}

	switch v := mf.Properties["photo"][0].(type) {
	case map[string]string:
		url = v["value"]
	case string:
		url = v
	}
	return
}

func parsePName(mf *microformats.Microformat) (name string) {
	if len(mf.Properties["name"]) < 1 {
		return
	}
	switch v := mf.Properties["name"][0].(type) {
	case map[string]string:
		name = v["value"]
	case string:
		name = v
	}
	return
}

func getHcard(link string) *hcard {
	hc, err := fetchHcard(link)
	if err != nil {
		h := hcard{}
		return &h
	}
	return hc
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
	hc := getHcard(req.Form["url"][0])

	js, err := json.Marshal(hc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
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
	hc := getHcard(req.Form["url"][0])

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

	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeContent(w, req, "", t, bytes.NewReader(bb))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/hcard", serveHcard)
	http.HandleFunc("/api/photo", servePhoto)

	http.ListenAndServe(":"+port, nil)
}
