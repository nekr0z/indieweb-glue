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

// Package pageinfo provides handling for information about internet pages.
package pageinfo

import (
	"net/http"
	"net/url"
	"strings"

	"evgenykuznetsov.org/go/indieweb-glue/internal/og"
	"github.com/PuerkitoBio/goquery"
	"willnorris.com/go/microformats"
)

// Info represents information about page
type Info struct {
	Title       string `json:"title,omitempty"`
	Image       string `json:"image,omitempty"`
	Description string `json:"description,omitempty"`
}

// Fetch fetches the page at URI and returns Info
func Fetch(uri string) (*Info, *http.Header, error) {
	u, err := url.Parse(uri)
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

	d, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, nil, err
	}

	pi := FromDocument(d)

	return &pi, &res.Header, nil
}

// FromDocument returns Info properties from a document
func FromDocument(d *goquery.Document) Info {
	o, _ := og.FromDocument(d)

	getTitle := []func(*goquery.Document) string{
		mfTitle,
		func(*goquery.Document) string { return o.Title },
		func(d *goquery.Document) string { return d.Find("title").Text() },
	}

	var title string
	for _, get := range getTitle {
		title = get(d)
		if len(title) != 0 {
			break
		}
	}

	getDescription := []func(*goquery.Document) string{
		mfDesc,
		func(*goquery.Document) string { return o.Description },
		wikiFirstPara,
	}

	var desc string
	for _, get := range getDescription {
		desc = get(d)
		if len(desc) != 0 {
			break
		}
	}

	return Info{
		Title:       title,
		Description: desc,
		Image:       o.Image,
	}
}

// wikiFirstPara returns the first paragraph of text if d is a wiki page
func wikiFirstPara(d *goquery.Document) string {
	gen, ok := d.Find("meta[name=\"generator\"]").Attr("content")
	if !ok {
		return ""
	}
	if !strings.HasPrefix(gen, "MediaWiki") {
		return ""
	}
	return d.Find(".mw-parser-output").Find("p").First().Text()
}

// mfDesc returns the description of a page that has microformats on it.
func mfDesc(d *goquery.Document) string {
	data := microformats.ParseNode(d.Get(0), nil)
	for _, item := range data.Items {
		if item.ID == "content" {
			n := item.Properties["summary"]
			return getString(n)
		}
	}
	return ""
}

// mfTitle returns the title of a page that has microformats on it.
func mfTitle(d *goquery.Document) string {
	data := microformats.ParseNode(d.Get(0), nil)
	for _, item := range data.Items {
		if item.ID == "content" {
			n := item.Properties["name"]
			return getString(n)
		}
	}
	return ""
}

// getString returns a string value nested in interface{}
func getString(n interface{}) string {
	switch v := n.(type) {
	case string:
		return v
	case []interface{}:
		if len(v) < 1 {
			return ""
		}
		return getString(v[0])
	default:
		return ""
	}
}
