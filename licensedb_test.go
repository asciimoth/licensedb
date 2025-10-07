package licensedb_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/asciimoth/licensedb"
)

func Test_Normalise(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GPL3+", "GPL-3.0-or-later"},
		{"GPL2+", "GPL-2.0-or-later"},
		{"gPl3", "GPL-3.0"},
		{"GpL2", "GPL-2.0"},
		{"mIt", "MIT"},
		{
			"asl20 oR gPl-3.0-wIth-autOconf-excEption",
			"Apache-2.0 OR GPL-3.0-or-later WITH Autoconf-exception-3.0",
		},
		{
			"nunit bsd fdsfsadf GpL2 GpL3+",
			"nunit BSD fdsfsadf GPL-2.0 GPL-3.0-or-later",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := licensedb.Normalise(tc.in)
			if got != tc.want {
				t.Fatalf("Normalise(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_ToShortForms(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"GPL-3.0-only", []string{
			"GPL",
			"GPL-3",
			"GPL-3.0",
			"GPL3",
			"GPL3.0",
		}},
		{"GPL-3.0-or-later", []string{
			"GPL",
			"GPL-3",
			"GPL-3.0",
			"GPL-3.0-or",
			"GPL3",
			"GPL3.0",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := licensedb.ToShortForms(tc.in)
			slices.Sort(got)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ShortForms(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_Extract(t *testing.T) {
	tests := []struct {
		in         string
		licenses   []string
		exceptions []string
		ambiguous  []string
		unknown    []string
	}{
		{"", []string{}, []string{}, []string{}, []string{}},
		{
			"bsd  fdsfsadf GpL2  Or  \n  aSl20  aNd  gPl-3.0-wIth-autOconf-excEption",
			[]string{"Apache-2.0", "GPL-3.0-or-later"},
			[]string{"Autoconf-exception-3.0"},
			[]string{"BSD", "GPL-2.0"},
			[]string{"fdsfsadf"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			l, e, a, u := licensedb.Extract(tc.in)
			same := reflect.DeepEqual(l, tc.licenses) &&
				reflect.DeepEqual(e, tc.exceptions) &&
				reflect.DeepEqual(a, tc.ambiguous) &&
				reflect.DeepEqual(u, tc.unknown)
			if !same {
				t.Fatalf(
					"Extract(%v) = %v %v %v %v; want %v %v %v %v",
					tc.in,
					l, e, a, u,
					tc.licenses, tc.exceptions, tc.ambiguous, tc.unknown,
				)
			}
		})
	}
}

func Test_AreMatching(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want bool
	}{
		{"", "", true},
		{"a", "b", false},
		{"MIT", "GPL", false},
		{"GPL", "GPL", true},
		{"GPL2", "GPL", true},
		{"GPL", "GPL3", true},
		{"GPL3", "GPL2", false},
		{"GPL3", "GPL3+", true},
		{"GPL3 MIT", "MIT GPL3+", true},
		{"MIT OR GPL-3.0-with-gcc-exception", "GPL3+ MIT", true},
		{"MIT OR GPL-3.0-with-gcc-exception", "GPL3+ MIT GCC-exception-3.1", true},
		{"MIT OR GPL-3.0-with-gcc-exception", "GPL3+ MIT Autoconf-exception-3.0", false},
	}

	for _, tc := range tests {
		t.Run(tc.a+" "+tc.b, func(t *testing.T) {
			t.Parallel()

			got := licensedb.AreMatching(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("AreMatching(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
