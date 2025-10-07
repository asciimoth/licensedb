package internal_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/asciimoth/licensedb/internal"
)

func ExampleGetText() {
	fmt.Println(internal.GetText("GPL-3.0-only"))
}

func Test_TokenToCanonical(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GPL3+", "GPL-3.0-or-later"},
		{"GPL2+", "GPL-2.0-or-later"},
		{"gPl3", "GPL-3.0"},
		{"GpL2", "GPL-2.0"},
		{"mIt", "MIT"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := internal.TokenToCanonical(tc.in)
			if got != tc.want {
				t.Fatalf("TokenToCanonical(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_CanonicalToGlobs(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"GPL-3.0-only", []string{
			"GPL",
			"GPL-3.0",
			"GPL-3.",
			"GPL-3.0.0",
			"GPL-3",
			"GPL-",
			"GPL3",
			"GPL3.0",
		}},
		{"GPL-3.0-or-later", []string{
			"GPL",
			"GPL-3.0",
			"GPL-3.0-or",
			"GPL-3.",
			"GPL-3.0.0",
			"GPL-3",
			"GPL-",
			"GPL3",
			"GPL3.0",
		}},
		{"Autoconf-exception-generic-3.0", []string{
			"Autoconf",
			"Autoconf-exception",
			"Autoconf-exception-generic",
			"Autoconf-exception-generic-3.",
			"Autoconf-exception-generic-3.0.0",
			"Autoconf-exception-generic-3",
			"Autoconf-exception-generic-",
			"Autoconf-exception-generic-3.0",
			"Autoconf-exception-generic3",
			"Autoconf-exception-generic3.0",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := internal.CanonicalToGlobs(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("CanonicalToGlobs(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_CanonicalToAllForms(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"GPL-3.0-or-later", []string{
			"gpl-3.0-or-later",
			"gpl-3.0+",
			"gpl-3+",
			"gpl3+",
			"gpl-3.00+",
		}},
		{"Autoconf-exception-generic-3.0", []string{
			"autoconf-exception-generic-3.0",
			"autoconf-exception-generic-3",
			"autoconf-exception-generic3",
			"autoconf-exception-generic-3.00",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := internal.CanonicalToAllForms(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("CanonicalToAllForms(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_DedupInPlace(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil slice", nil, nil},
		{"empty slice", []string{}, []string{}},
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"all duplicates", []string{"x", "x", "x"}, []string{"x"}},
		{
			"mixed case-sensitive",
			[]string{"apple", "Banana", "apple", "banana", "Cherry", "banana"},
			[]string{"apple", "Banana", "banana", "Cherry"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// create an input copy for the test, preserving nil vs empty distinction
			var in []string
			if tc.in != nil {
				in = make([]string, len(tc.in))
				copy(in, tc.in)
			} else {
				in = nil
			}

			got := internal.DedupInPlace(in)

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("DedupInPlace(%v) = %v; want %v", tc.in, got, tc.want)
			}

			// If input was non-nil and returned slice is non-empty, ensure the returned slice is a prefix
			// of the original backing array (i.e., we reused the backing array).
			if in != nil && len(got) > 0 {
				if !reflect.DeepEqual(in[:len(got)], got) {
					t.Fatalf("returned slice not a prefix of original backing array: input after call = %v, returned = %v", in, got)
				}
				if &in[0] != &got[0] {
					t.Fatalf("expected returned slice to share backing array with input")
				}
			}
		})
	}
}

func Test_ToShort(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GPL-3.0-or-later MiT", "GPL MIT"},
		{"GPL-3.0-or-later GPL-2.0-only or mIt", "GPL3 GPL2 OR MIT"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := internal.ToShort(tc.in)
			if got != tc.want {
				t.Fatalf("ToShort(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func Test_AreTokensMatching(t *testing.T) {
	// internal.DebugPrintGlobs()
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
	}

	for _, tc := range tests {
		t.Run(tc.a+" "+tc.b, func(t *testing.T) {
			t.Parallel()

			got := internal.AreTokensMatching(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("AreTokensMatching(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func Test_AreListsMatching(t *testing.T) {
	tests := []struct {
		a    []string
		b    []string
		want bool
	}{
		{[]string{""}, []string{""}, true},
		{[]string{"a"}, []string{"b"}, false},
		{[]string{"MIT"}, []string{"GPL"}, false},
		{[]string{"GPL"}, []string{"GPL"}, true},
		{[]string{"GPL2"}, []string{"GPL"}, true},
		{[]string{"GPL"}, []string{"GPL3"}, true},
		{[]string{"GPL3"}, []string{"GPL2"}, false},
		{[]string{"GPL3"}, []string{"GPL3+"}, true},
		{[]string{"GPL3", "MIT"}, []string{"GPL3+", "MIT"}, true},
		{[]string{"GPL3", "MIT"}, []string{"MIT", "GPL3+"}, true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.a, tc.b), func(t *testing.T) {
			t.Parallel()

			got := internal.AreListsMatching(tc.a, tc.b)
			if got != tc.want {
				t.Fatalf("AreTokensMatching(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func Test_SeparateTokenList(t *testing.T) {
	tests := []struct {
		tokens     []string
		licenses   []string
		exceptions []string
	}{
		{[]string{}, []string{}, []string{}},
		{
			[]string{"MIT", "GPL-3.0-or-later", "with", "GCC-exception-3.1"},
			[]string{"MIT", "GPL-3.0-or-later"},
			[]string{"GCC-exception-3.1"},
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprint(tc.tokens), func(t *testing.T) {
			t.Parallel()

			licenses, exceptions := internal.SeparateTokenList(tc.tokens)

			if !(reflect.DeepEqual(licenses, tc.licenses) &&
				reflect.DeepEqual(exceptions, tc.exceptions)) {
				t.Fatalf(
					"SeparateTokenList(%v) = %v %v; want %v %v",
					tc.tokens,
					licenses, exceptions,
					tc.licenses, tc.exceptions,
				)
			}
		})
	}
}

func Test_GlobToFirstMatch(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GPL", "GPL-1.0-only"},
		{"GPL3", "GPL-3.0-389-ds-base-exception"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got := internal.GlobToFirstMatch(tc.in)
			if got != tc.want {
				t.Fatalf("GlobToFirstMatch(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}
