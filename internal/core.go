package internal

import (
	"archive/zip"
	"bytes"
	"io"
	"slices"
	"strconv"
	"strings"
)

//go:generate go run genembed.go -url=https://github.com/spdx/license-list-data/archive/refs/tags/v3.27.0.zip -name=spdx3.27.0.zip

var (
	Files     map[string]*zip.File
	Filenames []string
	Globs     map[string][]string
	Canonical map[string]string
)

var (
	Keywords = []string{"WITH", "AND", "OR"}
	// Map of Deprecated IDs to expressions
	// Result of mapping may be ambiguous
	Deprecated = map[string][]string{
		"gpl-3.0+": {"gpl-3.0-or-later"},
		"gpl-2.0+": {"gpl-2.0-or-later"},
		"gpl-3.0-with-autoconf-exception": {
			"gpl-3.0-or-later", "with", "autoconf-exception-3.0",
		},
		"gpl-3.0-with-gcc-exception": {
			"gpl-3.0-or-later", "with", "gcc-exception-3.1",
		},
	}
	Aliases = []struct{ from, to string }{
		{"gpl3", "GPL-3.0"},
		{"gpl-3", "GPL-3.0"},
		{"gpl2", "GPL-2.0"},
		{"gpl-2", "GPL-2.0"},
		// nixpkgs
		{"asl20", "Apache-2.0"},
		{"asl11", "Apache-1.1"},
	}
	// TODO:
	ExceptionsList = []string{
		"GNU-compiler-exception",
		"GNOME-examples-exception",
		"Autoconf-exception-generic",
		"Autoconf-exception-generic-3.0",
		"Autoconf-exception-macro",
		"Autoconf-exception-2.0",
		"Autoconf-exception-3.0",
		"GCC-exception-2.0-note",
		"GCC-exception-2.0",
		"GCC-exception-3.1",
	}
)

func GetText(name string) *string {
	f, ok := Files[name]
	if !ok {
		return nil
	}
	rc, err := f.Open()
	if err != nil {
		// There should not be errors while working with embedded archive
		panic(err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		// There should not be errors while working with embedded archive
		panic(err)
	}
	strdata := string(data)
	return &strdata
}

func List() []string {
	out := make([]string, len(Filenames))
	copy(out, Filenames)
	return out
}

func initFiles() {
	r := bytes.NewReader(archive)
	zr, err := zip.NewReader(r, int64(len(archive)))
	if err != nil {
		// There should not be errors while working with embedded archive
		panic(err)
	}
	Files = make(map[string]*zip.File, len(zr.File))
	Filenames = make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		Files[f.Name] = f
		Filenames = append(Filenames, f.Name)
	}
}

// HyphenPrefixes returns cumulative prefixes of s split by '-' but
// excluding the full original string.
// Example: "a-b-c-d" -> ["a", "a-b", "a-b-c"]
func HyphenPrefixes(s string) []string {
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0)
	for i, r := range s {
		if r == '-' && i > 0 { // skip a leading '-' which would produce an empty prefix
			out = append(out, s[:i])
		}
	}
	return out
}

// DedupInPlace removes duplicates in-place and returns a subslice of the
// original backing array with duplicates removed, preserving the first occurrence order.
// This reuses memory (no new slice allocation for elements), O(n) time, O(n) map.
func DedupInPlace(ss []string) []string {
	if len(ss) == 0 {
		return ss
	}
	seen := make(map[string]struct{}, len(ss))
	j := 0
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ss[j] = s
		j++
	}
	return ss[:j]
}

// Produce all non-canonical forms for given canonocal one
func CanonicalToAllForms(canonical string) (forms []string) {
	canonical = strings.ToLower(canonical)
	forms = []string{canonical}
	if strings.Contains(canonical, "-or-later") {
		if strings.Contains(canonical, "-or-later-") {
			forms = append(forms, strings.ReplaceAll(canonical, "-or-later-", "+"))
		} else {
			forms2 := CanonicalToAllForms(strings.TrimSuffix(canonical, "-or-later"))
			for _, form := range forms2 {
				forms = append(forms, form+"+")
			}
		}
	}
	if strings.HasSuffix(canonical, ".0") {
		forms = append(
			forms, CanonicalToAllForms(strings.TrimSuffix(canonical, ".0"))...,
		)
	}
	for i := range 10 {
		suff := strconv.Itoa(i)
		if strings.HasSuffix(canonical, suff) {
			forms = append(
				forms, strings.TrimSuffix(canonical, "-"+suff)+suff,
			)
		}
	}
	for i := 15; i < 100; i += 10 {
		suff := strconv.FormatFloat(float64(i)/10.0, 'e', -1, 64)
		if strings.HasSuffix(canonical, suff) {
			forms = append(
				forms, strings.TrimSuffix(canonical, "-"+suff)+suff,
			)
		}
	}
	forms = DedupInPlace(forms)
	return
}

func initCanonical() {
	Canonical = make(map[string]string)
	for _, kw := range Keywords {
		Canonical[strings.ToLower(kw)] = kw
	}
	for _, alias := range Aliases {
		Canonical[alias.from] = alias.to
	}
	for _, file := range Filenames {
		for _, form := range CanonicalToAllForms(file) {
			Canonical[form] = file
		}
	}
}

func addGlob(k, v string) {
	_, ok := Globs[k]
	if ok {
		Globs[k] = append(Globs[k], v)
	} else {
		Globs[k] = []string{v}
	}
}

func CanonicalToGlobs(canonical string) (globs []string) {
	globs = make([]string, 0)

	globs = append(globs, HyphenPrefixes(canonical)...)

	// "-with" handling
	if strings.Contains(canonical, "-with") {
		parts := strings.SplitN(canonical, "-with", 2)
		globs = append(globs, CanonicalToGlobs(parts[0])...)
	}

	// suffix list handling
	for _, suffix := range []string{"-only", "-or-later", "-exception", "-note"} {
		if strings.HasSuffix(canonical, suffix) {
			trimmed := strings.TrimSuffix(canonical, suffix)
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		}
	}

	// deprecated_ prefix
	if strings.HasPrefix(canonical, "deprecated_") {
		trimmed := strings.TrimPrefix(canonical, "deprecated_")
		globs = append(globs, trimmed)
		globs = append(globs, CanonicalToGlobs(trimmed)...)
	}

	// numeric-driven transformations (0..9)
	for i := range 10 {
		si := strconv.Itoa(i)

		// endswith("-i.0")
		if strings.HasSuffix(canonical, "-"+si+".0") {
			trimmed := strings.TrimSuffix(canonical, "-"+si+".0")
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		}

		// endswith(i)
		if strings.HasSuffix(canonical, si) {
			trimmed := strings.TrimSuffix(canonical, si)
			globs = append(globs, trimmed)
			globs = append(globs, canonical+".0")
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		}

		// contains("-i") else contains "i"
		if strings.Contains(canonical, "-"+si) {
			trimmed := strings.SplitN(canonical, "-"+si, 2)[0]
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		} else if strings.Contains(canonical, si) {
			trimmed := strings.SplitN(canonical, si, 2)[0]
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		}

		// endswith(".i")
		suffix := "." + si
		if strings.HasSuffix(canonical, suffix) {
			trimmed := strings.TrimSuffix(canonical, suffix)
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)
		}

		// endswith("-i") special handling (two trimmed variants)
		suffix = "-" + si
		if strings.HasSuffix(canonical, suffix) {
			trimmed := strings.TrimSuffix(canonical, suffix) + si
			globs = append(globs, trimmed)
			globs = append(globs, CanonicalToGlobs(trimmed)...)

			trimmed2 := strings.TrimSuffix(canonical, suffix)
			globs = append(globs, trimmed2)
			globs = append(globs, CanonicalToGlobs(trimmed2)...)
		}
	}

	globs = DedupInPlace(globs)
	return
}

func initGlobs() {
	Globs = make(map[string][]string)
	for _, file := range Filenames {
		for _, glob := range CanonicalToGlobs(file) {
			addGlob(glob, file)
		}
	}
	for glob, _ := range Globs {
		Globs[glob] = DedupInPlace(Globs[glob])
	}
}

func initDeprecated() {
	for _, file := range Filenames {
		if !strings.HasPrefix(file, "deprecated_") {
			continue
		}
		clean := strings.TrimPrefix(file, "deprecated_")
		_, ok := Deprecated[clean]
		if ok {
			continue
		}
		Deprecated[clean] = []string{file}
	}
}

func init() {
	initFiles()
	initDeprecated()
	initGlobs()
	initCanonical()
}

func TokenToCanonical(token string) string {
	token = strings.ToLower(token)
	if c, ok := Canonical[token]; ok {
		return strings.TrimPrefix(c, "deprecated_")
	}
	upper := strings.ToUpper(token)
	if _, ok := Globs[upper]; ok {
		return upper
	}
	trimmed := strings.TrimSuffix(token, "+")
	if c, ok := Canonical[trimmed]; ok {
		return strings.TrimPrefix(c, "deprecated_") + "+"
	}
	upper = strings.ToUpper(trimmed)
	if _, ok := Globs[upper]; ok {
		return upper + "+"
	}
	return token
}

func TokensToCanonical(tokens []string) []string {
	canon := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token == "" || token == " " {
			continue
		}
		token := strings.ToLower(token)
		depr, ok := Deprecated[token]
		if ok {
			canon = append(canon, TokensToCanonical(depr)...)
			continue
		}
		canon = append(canon, TokenToCanonical(token))
	}
	return canon
}

func Tokenise(text string) []string {
	for _, banned := range []string{"\t", "\n", "\r"} {
		text = strings.ReplaceAll(text, banned, " ")
	}
	return TokensToCanonical(strings.Split(text, " "))
}

func TokensToShort(tokens []string) map[string]string {
	mapping := make(map[string]string)
	for _, token := range tokens {
		if slices.Contains(Keywords, token) {
			continue
		}
		short := token
		for glob, matches := range Globs {
			if len(glob) >= len(short) {
				continue
			}
			if !slices.Contains(matches, token) {
				continue
			}
			ok := true
			for _, another_token := range tokens {
				if another_token == token {
					continue
				}
				if slices.Contains(Keywords, another_token) {
					continue
				}
				if slices.Contains(matches, another_token) {
					ok = false
					break
				}
			}
			if ok {
				short = glob
			}
		}
		mapping[token] = short
	}
	return mapping
}

func ToShort(text string) string {
	tokens := Tokenise(text)
	mapping := TokensToShort(tokens)
	for i := range len(tokens) {
		if slices.Contains(Keywords, tokens[i]) {
			continue
		}
		tokens[i] = mapping[tokens[i]]
	}
	text = strings.Join(tokens, " ")
	return strings.Join(strings.Fields(text), " ")
}

func GetGlobs(token string) []string {
	result := []string{token}
	if g, ok := Globs[token]; ok {
		result = append(result, g...)
	}
	return result
}

func AreTokensMatching(a, b string) bool {
	if a == b {
		return true
	}
	if a+"+" == b || a == b+"+" {
		return true
	}
	variantsA := GetGlobs(a)
	variantsB := GetGlobs(b)
	for _, va := range variantsA {
		if slices.Contains(variantsB, va) {
			return true
		}
	}
	return false
}

func SeparateTokenList(tokens []string) (licenses, exceptions []string) {
	licenses = make([]string, 0, len(tokens))
	exceptions = make([]string, 0, len(tokens))
	for _, token := range tokens {
		upper := strings.ToUpper(token)
		if token == "" || token == " " || slices.Contains(Keywords, upper) {
			continue
		}
		if slices.Contains(ExceptionsList, token) {
			exceptions = append(exceptions, token)
			continue
		}
		licenses = append(licenses, token)
	}
	return
}

func AreListsMatching(a, b []string) bool {
	for _, ea := range a {
		matching := false
		for _, eb := range b {
			if AreTokensMatching(ea, eb) {
				matching = true
				break
			}
		}
		if !matching {
			return false
		}
	}
	return true
}

func areTokensListsMatching(a, b []string) bool {
	al, ae := SeparateTokenList(a)
	bl, be := SeparateTokenList(b)
	if len(ae) > 0 && len(be) > 0 && !AreListsMatching(ae, be) {
		return false
	}
	return AreListsMatching(al, bl)
}

func AreTokensListsMatchingSwap(a, b []string) bool {
	return areTokensListsMatching(a, b) && areTokensListsMatching(b, a)
}

func GlobToFirstMatch(glob string) string {
	if _, ok := Files[glob]; ok {
		return glob
	}
	globs, ok := Globs[glob]
	if !ok {
		return glob
	}
	for _, g := range globs {
		r := GlobToFirstMatch(g)
		if r != glob {
			return r
		}
	}
	return glob
}
