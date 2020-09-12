package main

import (
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	mf "willnorris.com/go/microformats"
)

func getRepresentativeHcard(r *http.Response) (m *mf.Microformat) {
	defer r.Body.Close()
	doc, err := goquery.NewDocumentFromReader(r.Body)
	if err != nil {
		return
	}

	hcards := getHcards(doc, r.Request.URL)

	// check 1 (first h-card where uid == url == page URL)
	for _, hc := range hcards {
		if matchUrlUid(hc, r.Request.URL) {
			return hc
		}
	}

	// check 2 (first h-card where url has a rel=me relation)
	d := mf.ParseNode(doc.Get(0), r.Request.URL)
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
		if matchURLs(parseProperty(hcards[0], "url"), r.Request.URL.String()) {
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
