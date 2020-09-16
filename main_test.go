package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFetchHcard(t *testing.T) {
	tests := map[string]struct {
		link string
		want string
	}{
		"evgeny": {"", "%s/img/avatar.jpg"},
		"tim":    {"/tim.html", ""},
		"aaron":  {"/aaron.html", "%s/images/profile.jpg"},
		"ruxton": {"/ignition.html", "https://secure.gravatar.com/avatar/8401de9afbdfada34ca21681a2394340?s=125&d=default&r=g"},
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

			want := fmt.Sprintf(tc.want, s.URL)
			if strings.Contains(want, "%!(EXTRA string") {
				want = tc.want
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

	want := fmt.Sprintf(`{"source":"%s","pname":"Евгений Кузнецов","uphoto":"%s/img/avatar.jpg"}`, ms.URL, ms.URL)
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
func TestCopyHeader(t *testing.T) {
	r := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Etag": []string{`"1553c-5a234afb92e92"`},
		},
	}
	w := httptest.NewRecorder()

	r.Header.Add("cache-control", "max-age=2592000")
	r.Header.Add("cache-control", "public")

	copyHeader(r, w, "Etag")

	res := w.Result()
	h := res.Header.Values("Etag")
	if len(h) != 1 {
		t.Fatalf("etag header length: want %d, got %d", 1, len(h))
	}
	if h[0] != `"1553c-5a234afb92e92"` {
		t.Fatalf("etag header: want \"1553c-5a234afb92e92\", got %s", h[1])
	}
}
