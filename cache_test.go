// Copyright (C) 2021 Evgeny Kuznetsov (evgeny@kuznetsov.md)
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
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"net/http"
	"testing"
)

func TestCanCache(t *testing.T) {
	tests := map[string]struct {
		cc   []string
		want bool
	}{
		"none":            {[]string{}, true},
		"public, no spec": {[]string{"public"}, true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			h := make(http.Header)
			for _, c := range tc.cc {
				h.Add("Cache-Control", c)
			}

			got, _ := canCache(h)
			if got != tc.want {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}
