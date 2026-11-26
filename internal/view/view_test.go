// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package view_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/view"
)

func TestStelliaPoints(t *testing.T) {
	stellia := []game.Stellium{{ID: 1, X: 1, Y: 2, Z: 3}, {ID: 2, X: -4, Y: 0, Z: 5, HomeStellium: true}, {ID: 3, X: 0, Y: 0, Z: -1}}
	for _, tc := range []struct {
		name    string
		stars   []game.Star
		want    []view.StelliumPoint
		wantErr bool
	}{
		{
			name:  "counts stars per stellium",
			stars: []game.Star{{ID: 1, StelliumID: 1, SeqNo: 1}, {ID: 2, StelliumID: 1, SeqNo: 2}, {ID: 3, StelliumID: 3, SeqNo: 1}},
			want: []view.StelliumPoint{
				{ID: 1, X: 1, Y: 2, Z: 3, Stars: 2},
				{ID: 2, X: -4, Y: 0, Z: 5, Stars: 0, Home: true},
				{ID: 3, X: 0, Y: 0, Z: -1, Stars: 1},
			},
		},
		{
			name:    "rejects unknown stellium",
			stars:   []game.Star{{ID: 1, StelliumID: 9, SeqNo: 1}},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := view.StelliaPoints(stellia, tc.stars)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("got nil error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestStelliaPage(t *testing.T) {
	stellia := []game.Stellium{{ID: 7, X: -3, Y: 4, Z: 12, HomeStellium: true}, {ID: 8, X: 1, Y: 1, Z: 1}}
	stars := []game.Star{{ID: 1, StelliumID: 7, SeqNo: 1}, {ID: 2, StelliumID: 8, SeqNo: 1}}
	for _, tc := range []struct {
		name string
		code string
		want []string
	}{
		{
			name: "embeds data and title",
			code: "alpha",
			want: []string{
				"<title>Stellia of alpha</title>",
				`const POINTS = [{"id":7,"x":-3,"y":4,"z":12,"stars":1,"home":true},{"id":8,"x":1,"y":1,"z":1,"stars":1,"home":false}];`,
				"1 is a home Stellium.",
			},
		},
		{
			name: "escapes the game code",
			code: "<b>",
			want: []string{"<title>Stellia of &lt;b&gt;</title>"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder
			if err := view.Stellia(&sb, tc.code, stellia, stars); err != nil {
				t.Fatalf("got error %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(sb.String(), want) {
					t.Errorf("page lacks %q", want)
				}
			}
		})
	}
}
