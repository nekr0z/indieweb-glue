// Copyright (C) 2022 Evgeny Kuznetsov (evgeny@kuznetsov.md)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along tihe this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"evgenykuznetsov.org/go/indieweb-glue/internal/hcard"
	"evgenykuznetsov.org/go/indieweb-glue/internal/og"
	"evgenykuznetsov.org/go/indieweb-glue/internal/pageinfo"
	"github.com/memcachier/mc/v3"
)

//go:embed tpl/*
var tpl embed.FS

var websiteUrl string

func calculateExpiration(h, hd http.Header) (bool, time.Time) {
	ok, expH := canCache(h)
	if !ok {
		return false, time.Now()
	}
	ok, expHd := canCache(hd)
	if !ok {
		return false, time.Now()
	}

	if expH.Before(expHd) {
		return true, expH
	}
	return true, expHd
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
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

func getPhoto(c cache, link string) ([]byte, map[string][]string, error) {
	key := "photo=" + link
	content, exp := c.get(key)
	if content != nil {
		fmt.Printf("photo %s cache hit\n", link)
		hd := map[string][]string{
			"Cache-Control": {"public"},
			"Expires":       {exp.Format(time.RFC1123)},
		}
		return content, hd, nil
	}

	bb := []byte{}
	hd := http.Header{}

	res, err := http.Get(link)
	if err != nil {
		return bb, hd, err
	}
	defer res.Body.Close()

	bb, err = io.ReadAll(res.Body)
	if err != nil {
		return bb, hd, err
	}

	hd = res.Header.Clone()

	if ok, exp := canCache(hd); ok {
		c.set(key, bb, exp)
		fmt.Printf("%s cached until %s\n", key, exp.Format(time.RFC1123))
	} else {
		fmt.Printf("%s not cached\n", key)
	}
	return bb, hd, nil
}

func getModTime(hd http.Header) time.Time {
	lm, ok := hd["Last-Modified"]
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

// serveJSON serves JSON response returned from getter, caches it as needed
func serveJSON(c cache, cachePrefix string, g getter) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(req.Form["url"]) < 1 {
			http.Error(w, "no URL specified", http.StatusBadRequest)
			return
		}

		content, hd := getJSON(c, cachePrefix, req.Form["url"][0], g)

		if content == nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		setResponseHeaders(w, hd)

		if string(content) == `{}` {
			http.Error(w, "no appropriate info at URL", http.StatusNotFound)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(content)
	}
}

// getJSON gets JSON response returned from getter, caches it as needed
func getJSON(c cache, cachePrefix, link string, g getter) (content []byte, hd map[string][]string) {
	key := fmt.Sprintf("%s=%s", cachePrefix, link)
	content, exp := c.get(key)
	if content != nil {
		fmt.Printf("%s %s cache hit\n", cachePrefix, link)
		hd = map[string][]string{
			"Cache-Control": {"public"},
			"Expires":       {exp.Format(time.RFC1123)},
		}
	} else {
		content, hd = g(link)
		if ok, exp := canCache(hd); ok && content != nil {
			c.set(key, content, exp)
			fmt.Printf("%s cached until %s\n", key, exp.Format(time.RFC1123))
		} else {
			fmt.Printf("%s not cached\n", key)
		}
	}
	return
}

func serveInfo(w http.ResponseWriter, req *http.Request) {
	fp := path.Join("tpl", "index.html")
	tmpl, err := template.ParseFS(tpl, fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public")
	w.Header().Add("Cache-Control", "max-age=3600")

	d := struct{ Addr string }{websiteUrl}
	if err := tmpl.Execute(w, d); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func servePhoto(c cache) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(req.Form["url"]) < 1 {
			http.Error(w, "no URL specified", http.StatusBadRequest)
			return
		}
		js, hchd := getJSON(c, "hcard", req.Form["url"][0], getHcard)
		hc := hcard.HCard{}
		if err := json.Unmarshal(js, &hc); err != nil {
			http.Error(w, "no hcard", http.StatusNotFound)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")

		if hc.Photo == "" {
			http.Error(w, "no photo", http.StatusNotFound)
			return
		}

		bb, hd, err := getPhoto(c, hc.Photo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if ok, exp := calculateExpiration(hchd, hd); ok {
			w.Header().Set("Expires", exp.Format(time.RFC1123))
			w.Header().Set("Cache-Control", "public")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}

		t := getModTime(hd)
		http.ServeContent(w, req, "", t, bytes.NewReader(bb))
	}
}

func cached(c cache, handler func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, exp := c.get(r.RequestURI)
		if content != nil {
			fmt.Printf("%s cache hit\n", r.RequestURI)
			w.Header().Set("Cache-Control", "public")
			w.Header().Set("Expires", exp.Format(time.RFC1123))
			w.Header().Set("Access-Control-Allow-Origin", "*")
			_, _ = w.Write(content)
		} else {
			re := httptest.NewRecorder()
			handler(re, r)

			content := re.Body.Bytes()
			res := re.Result()
			for k := range res.Header {
				copyHeader(res.Header, w, k)
			}
			w.WriteHeader(re.Code)

			if ok, exp := canCache(res.Header); ok {
				c.set(r.RequestURI, content, exp)
				fmt.Printf("%s cached until %s\n", r.RequestURI, exp.Format(time.RFC1123))
			} else {
				fmt.Printf("%s not cached\n", r.RequestURI)
			}

			_, _ = w.Write(content)
		}
	})
}

func main() {
	initSignalHandling()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	websiteUrl = os.Getenv("URL")
	if websiteUrl == "" {
		websiteUrl = "https://indieweb-glue.evgenykuznetsov.org"
	}

	var c cache
	mcPass := os.Getenv("MEMCACHIER_PASSWORD")
	mcSrv := os.Getenv("MEMCACHIER_SERVERS")
	mcUser := os.Getenv("MEMCACHIER_USERNAME")
	if mcPass != "" && mcSrv != "" && mcUser != "" {
		client := mc.NewMC(mcSrv, mcUser, mcPass)
		defer client.Quit()
		c = newMcCache(client)
		fmt.Println("using memcached")
	} else {
		c = newMemoryCache()
		fmt.Println("using memory cache")
	}

	http.HandleFunc("/api/hcard", serveJSON(c, "hcard", getHcard))
	http.HandleFunc("/api/opengraph", serveJSON(c, "og", getOG))
	http.HandleFunc("/api/pageinfo", serveJSON(c, "pageinfo", getPageInfo))
	http.HandleFunc("/api/photo", servePhoto(c))
	http.Handle("/", cached(c, serveInfo))

	_ = http.ListenAndServe(":"+port, nil)
}

func initSignalHandling() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("caught signal, terminating")
		os.Exit(0)
	}()
}

// getter takes an uri and returns a JSON-packed response for that uri
// (nil if marshaling failed), together with HTTP headers that may be of interest
type getter func(uri string) (js []byte, headers map[string][]string)

// getHcard is a getter for H-Cards
func getHcard(link string) ([]byte, map[string][]string) {
	hc, hd, err := hcard.Fetch(link)
	if err != nil {
		var hdr http.Header
		hc, hdr = hcard.Empty()
		hd = &hdr
	}
	content, err := json.Marshal(hc)
	if err != nil {
		fmt.Println("can't marshal hcard")
		return nil, *hd
	}
	return content, *hd
}

// getOG is a getter for OpenGraph
func getOG(link string) ([]byte, map[string][]string) {
	o, hd, err := og.Fetch(link)
	if err != nil {
		return []byte("{}"), nil
	}
	content, err := json.Marshal(o)
	if err != nil {
		fmt.Println("failed to marshal OG")
		return nil, *hd
	}
	return content, *hd
}

// getPageIngo is a getter for page information
func getPageInfo(link string) ([]byte, map[string][]string) {
	pi, hd, err := pageinfo.Fetch(link)
	if err != nil {
		return []byte("{}"), nil
	}
	content, err := json.Marshal(pi)
	if err != nil {
		fmt.Println("failed to marshal page information")
		return nil, *hd
	}
	return content, *hd
}
