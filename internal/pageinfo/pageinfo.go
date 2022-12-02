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

	o, err := og.FromDocument(d)
	if err != nil {
		return nil, nil, err
	}

	desc := o.Description

	if len(desc) == 0 {
		if gen, ok := d.Find("meta[name=\"generator\"]").Attr("content"); ok {
			if strings.HasPrefix(gen, "MediaWiki") {
				desc = d.Find(".mw-parser-output").Find("p").First().Text()
			}
		}
	}

	pi := Info{
		Title:       o.Title,
		Description: desc,
		Image:       o.Image,
	}

	return &pi, &res.Header, nil
}
