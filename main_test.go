package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestFetchHcard(t *testing.T) {
	tests := map[string]struct {
		link string
		want string
	}{
		"evgeny": {"", "/img/avatar.jpg"},
		"tim":    {"/tim.html", ""},
		"aaron":  {"/aaron.html", "/images/profile.jpg"},
	}

	fs := http.FileServer(http.Dir("testdata"))
	s := httptest.NewServer(fs)
	defer s.Close()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			hc, err := fetchHcard(fmt.Sprintf("%s%s", s.URL, tc.link))
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			var want string
			if tc.want != "" {
				want = fmt.Sprintf("%s%s", s.URL, tc.want)
			}
			got := hc.Photo

			if got != want {
				t.Fatalf("want %v, got %v", want, got)
			}
		})
	}
}

func TestServeHcard(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(serveHcard))
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

	want := fmt.Sprintf(`{"uphoto":"%s/img/avatar.jpg"}`, ms.URL)
	if string(b) != want {
		t.Fatalf("want %s, got %s", want, b)
	}
}

func TestServePhoto(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(servePhoto))
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
