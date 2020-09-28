// Copyright (C) 2020 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"time"

	"github.com/memcachier/mc/v3"
)

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

	bb, err = ioutil.ReadAll(res.Body)
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

func serveHcard(c cache) func(http.ResponseWriter, *http.Request) {
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
		hc, hd := getHcard(c, req.Form["url"][0])

		js, err := json.Marshal(hc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if string(js) == `{}` {
			http.Error(w, "no representative hcard at URL", http.StatusNotFound)
		}

		setResponseHeaders(w, hd)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(js)
	}
}

func serveInfo(w http.ResponseWriter, req *http.Request) {
	fp := path.Join("tpl", "index.html")
	tmpl, err := template.ParseFiles(fp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "public")
	w.Header().Add("Cache-Control", "max-age=3600")

	d := struct{ Addr string }{"https://indieweb-glue.herokuapp.com"}
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
		hc, hchd := getHcard(c, req.Form["url"][0])

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

		w.Header().Set("Access-Control-Allow-Origin", "*")

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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

	http.HandleFunc("/api/hcard", serveHcard(c))
	http.HandleFunc("/api/photo", servePhoto(c))
	http.Handle("/", cached(c, serveInfo))

	_ = http.ListenAndServe(":"+port, nil)
}
