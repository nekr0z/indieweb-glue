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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestServeHcard(t *testing.T) {
	c := newMemoryCache()
	s := httptest.NewServer(http.HandlerFunc(serveHcard(c)))
	defer s.Close()

	fs := http.FileServer(http.Dir("testdata"))
	ms := httptest.NewServer(fs)
	defer ms.Close()

	u, _ := url.Parse(s.URL)
	v := url.Values{}
	v.Add("url", ms.URL)
	u.RawQuery = v.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	want := fmt.Sprintf(`{"source":"%s","pname":"Евгений Кузнецов","uphoto":"%s/img/avatar.jpg"}`, ms.URL, ms.URL)
	if string(b) != want {
		t.Fatalf("want %s, got %s", want, b)
	}
}

func TestServeEmptyHcard(t *testing.T) {
	c := newMemoryCache()
	s := httptest.NewServer(http.HandlerFunc(serveHcard(c)))
	defer s.Close()

	fs := http.FileServer(http.Dir("testdata"))
	ms := httptest.NewServer(fs)
	defer ms.Close()

	u, _ := url.Parse(s.URL)
	v := url.Values{}
	v.Add("url", ms.URL+"/404.html")
	u.RawQuery = v.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("want %v, got %v", http.StatusNotFound, res.StatusCode)
	}
}

func TestServePhoto(t *testing.T) {
	c := newMemoryCache()
	s := httptest.NewServer(http.HandlerFunc(servePhoto(c)))
	defer s.Close()

	fs := http.FileServer(http.Dir("testdata"))
	ms := httptest.NewServer(fs)
	defer ms.Close()

	u, _ := url.Parse(s.URL)
	v := url.Values{}
	v.Add("url", ms.URL)
	u.RawQuery = v.Encode()

	res, err := http.Get(u.String())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	defer res.Body.Close()

	got, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	want, _ := ioutil.ReadFile("testdata/img/avatar.jpg")
	if len(got) != len(want) {
		t.Fatalf("want length of %d bytes, got %d", len(want), len(got))
	}
	for i, b := range got {
		if want[i] != b {
			t.Fatalf("want %s for byte no. %d, got %s", string(want[i]), i, string(b))
		}
	}
}
func TestCopyHeader(t *testing.T) {
	hd := map[string][]string{
		"Etag": {
			`"1553c-5a234afb92e92"`,
		},
		"Cache-Control": {
			"max-age=2592000",
			"public",
		},
	}
	w := httptest.NewRecorder()

	copyHeader(hd, w, "etag")
	copyHeader(hd, w, "cache-control")

	res := w.Result()
	h := res.Header.Values("Etag")
	if len(h) != 1 {
		t.Fatalf("etag header length: want %d, got %d", 1, len(h))
	}
	if h[0] != `"1553c-5a234afb92e92"` {
		t.Fatalf("etag header: want \"1553c-5a234afb92e92\", got %s", h[1])
	}

	h = res.Header.Values("Cache-Control")
	if len(h) != 2 {
		t.Fatalf("cache-control header length: want %d, got %d", 2, len(h))
	}
}
