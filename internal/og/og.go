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

// Package og (for OpenGraph) provides handling for OpenGraph information.
package og

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

// OpenGraph represents OpenGraph information
type OpenGraph struct {
	Title       string `json:"title"`
	Image       string `json:"image,omitempty"`
	Description string `json:"description,omitempty"`
}

// Fetch fetches the page at URI and returns OpenGraph info
func Fetch(uri string) (*OpenGraph, *http.Header, error) {
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

	title, ok := d.Find("meta[property=\"og:title\"]").Attr("content")
	if !ok {
		return nil, nil, fmt.Errorf("no opengraph title property found")
	}
	og := OpenGraph{Title: title}

	if image, ok := d.Find("meta[property=\"og:image\"]").Attr("content"); ok {
		og.Image = image
	}

	if desc, ok := d.Find("meta[property=\"og:description\"]").Attr("content"); ok {
		og.Description = desc
	}

	return &og, &res.Header, nil
}
