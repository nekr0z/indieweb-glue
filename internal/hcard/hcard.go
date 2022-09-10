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

package hcard

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	mf "willnorris.com/go/microformats"
)

// HCard represents a h-card
type HCard struct {
	Source string `json:"source,omitempty"`
	PName  string `json:"pname,omitempty"`
	Photo  string `json:"uphoto,omitempty"`
}

func getRepresentativeHcard(page io.Reader, url *url.URL) (m *mf.Microformat) {
	doc, err := goquery.NewDocumentFromReader(page)
	if err != nil {
		return
	}

	hcards := getHcards(doc, url)

	// check 1 (first h-card where uid == url == page URL)
	for _, hc := range hcards {
		if matchUrlUid(hc, url) {
			return hc
		}
	}

	// check 2 (first h-card where url has a rel=me relation)
	d := mf.ParseNode(doc.Get(0), url)
	if mm, ok := d.Rels["me"]; ok {
		for _, hc := range hcards {
			for _, me := range mm {
				if matchURLs(parseProperty(hc, "url"), me) {
					return hc
				}
			}
		}
	}

	// check 3 (single h-card and url == page URL)
	if len(hcards) == 1 {
		if matchURLs(parseProperty(hcards[0], "url"), url.String()) {
			return hcards[0]
		}
	}

	return
}

func getHcards(doc *goquery.Document, u *url.URL) (hcards []*mf.Microformat) {
	nodes := doc.Find(".h-card").Nodes

	for _, n := range nodes {
		d := mf.ParseNode(n, u)
		for _, i := range d.Items {
			for _, t := range i.Type {
				if t == "h-card" {
					hcards = append(hcards, i)
				}
			}
		}
	}

	return
}

func Fetch(link string) (*HCard, *http.Header, error) {
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

	var hc HCard
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

func Empty() (*HCard, map[string][]string) {
	h := HCard{}
	hd := map[string][]string{}
	return &h, hd
}

func parseProperty(m *mf.Microformat, property string) (value string) {
	if len(m.Properties[property]) < 1 {
		return
	}

	switch v := m.Properties[property][0].(type) {
	case map[string]string:
		value = v["value"]
	case string:
		value = v
	}
	return
}

func matchUrlUid(hc *mf.Microformat, u *url.URL) bool {
	uidString := parseProperty(hc, "uid")
	if uidString == "" {
		return false
	}

	urlString := parseProperty(hc, "url")

	if matchURLs(uidString, urlString) && matchURLs(urlString, u.String()) {
		return true
	}
	return false
}

func matchURLs(a, b string) bool {
	aUrl, err := url.Parse(a)
	if err != nil {
		return false
	}

	bUrl, err := url.Parse(b)
	if err != nil {
		return false
	}

	if aUrl.String() != bUrl.String() {
		return false
	}

	return true
}
