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
	"strings"
	"testing"
)

func TestFetchDescription(t *testing.T) {
	tests := map[string]struct {
		link string
		want string
	}{
		"wikipedia": {"/sedgewick.html", "Роберт Седжвик (род."},
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
