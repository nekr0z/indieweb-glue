package pageinfo

import (
	"net/http"

	"evgenykuznetsov.org/go/indieweb-glue/internal/og"
)

// Info represents information about page
type Info struct {
	Title       string `json:"title,omitempty"`
	Image       string `json:"image,omitempty"`
	Description string `json:"description,omitempty"`
}

// Fetch fetches the page at URI and returns Info
func Fetch(uri string) (*Info, *http.Header, error) {
	o, hd, err := og.Fetch(uri)
	if err != nil {
		return nil, nil, err
	}

	pi := Info{
		Title:       o.Title,
		Description: o.Description,
		Image:       o.Image,
	}

	return &pi, hd, nil
}
