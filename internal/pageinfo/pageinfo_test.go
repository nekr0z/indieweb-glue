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

package pageinfo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestFetchDescription(t *testing.T) {
	tests := map[string]struct {
		link string
		want string
	}{
		"wikipedia":     {"/sedgewick.html", "Роберт Седжвик (род."},
		"indieweb wiki": {"/person_mention.html", "person mention is a homepage"},
		"jamesg.blog":   {"/capjamesg.html", "Hello! I'm James"},
	}

	fs := http.FileServer(http.Dir("testdata"))
	s := httptest.NewServer(fs)
	defer s.Close()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pi, _, err := Fetch(fmt.Sprintf("%s%s", s.URL, tc.link))
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if !strings.HasPrefix(pi.Description, tc.want) {
				t.Fatalf("want %v..., got %v", tc.want, pi.Description)
			}
		})
	}
}

func TestDescription(t *testing.T) {
	tests := map[string]struct {
		filename string
		want     string
	}{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pi := piFromFile(t, tc.filename)
			if pi.Description != tc.want {
				t.Fatalf("want \"%v\", got \"%v\"", tc.want, pi.Description)
			}
		})
	}
}

func TestImage(t *testing.T) {
	tests := map[string]struct {
		link string
		want string
	}{
		"wikipedia":  {"/sedgewick.html", "https://upload.wikimedia.org/wikipedia/commons/d/d1/Robertsedgewick.jpg"},
		"u-featured": {"/james.html", "%s/assets/hovercard.png"},
	}

	fs := http.FileServer(http.Dir("testdata"))
	s := httptest.NewServer(fs)
	defer s.Close()

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pi, _, err := Fetch(fmt.Sprintf("%s%s", s.URL, tc.link))
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			want := fmt.Sprintf(tc.want, s.URL)
			if strings.Contains(want, "%!(EXTRA string") {
				want = tc.want
			}

			if pi.Image != want {
				t.Fatalf("want %v, got %v", want, pi.Image)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	tests := map[string]struct {
		filename string
		want     string
	}{
		"indieweb wiki": {"person_mention.html", "person mention"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			pi := piFromFile(t, tc.filename)
			if pi.Title != tc.want {
				t.Fatalf("want \"%v\", got \"%v\"", tc.want, pi.Title)
			}
		})
	}
}

func piFromFile(t *testing.T, filename string) Info {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}
	return FromDocument(d)
}
