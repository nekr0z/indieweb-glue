package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	"willnorris.com/go/microformats"
)

type hcard struct {
	Photo string `json:"u-photo,omitempty"`
}

func fetchHcard(link string) (*hcard, error) {
	url, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	if url.Scheme == "" {
		url.Scheme = "http"
	}

	res, err := http.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	d := microformats.Parse(res.Body, url)

	var hc hcard

	for _, i := range d.Items {
		for _, t := range i.Type {
			if t == "h-card" {
				hc.Photo = parsePhotoUrl(i)
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

func getHcard(link string) *hcard {
	hc, err := fetchHcard(link)
	if err != nil {
		h := hcard{}
		return &h
	}
	return hc
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

	w.Write([]byte("not yet implemented"))
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
