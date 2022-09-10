package hcard

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
			hc, _, err := Fetch(fmt.Sprintf("%s%s", s.URL, tc.link))
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
