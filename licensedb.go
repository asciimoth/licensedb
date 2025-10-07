package licensedb

import (
	"slices"
	"strings"

	"github.com/asciimoth/licensedb/internal"
)

// Normalise converts alternative forms of SPDX IDs in text to their normal form.
func Normalise(text string) string {
	// BUG: just `strings.Join(Tokenise(text), " ")` returns string with extra whitespaces
	text = strings.Join(internal.Tokenise(text), " ")
	return strings.Join(strings.Fields(text), " ")
}

// ToShortForms converts SPDX IDs in text to their alternative short names.
func ToShortForms(text string) []string {
	forms := []string{}
	for k, v := range internal.Globs {
		if !slices.Contains(v, text) {
			continue
		}
		if len(k) >= len(text) {
			continue
		}
		if strings.HasSuffix(k, "-") {
			continue
		}
		if strings.HasSuffix(k, ".") {
			continue
		}
		if strings.HasSuffix(k, ".0.0") {
			continue
		}
		forms = append(forms, k)
	}
	return forms
}

// Extract extratcs SPDX IDs from text expression.
func Extract(expr string) (licenses, exceptions, ambiguous, unknown []string) {
	tokens := internal.Tokenise(expr)
	licenses = make([]string, 0, len(tokens))
	exceptions = make([]string, 0, len(tokens))
	ambiguous = make([]string, 0, len(tokens))
	unknown = make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" || token == " " || slices.Contains(internal.Keywords, token) {
			continue
		}
		if _, ok := internal.Files[token]; !ok {
			if _, ok := internal.Globs[token]; ok {
				ambiguous = append(ambiguous, token)
				continue
			}
			unknown = append(unknown, token)
			continue
		}
		if slices.Contains(internal.ExceptionsList, token) {
			exceptions = append(exceptions, token)
			continue
		}
		licenses = append(licenses, token)
	}
	return
}

// AreMatching reports if two expressions contains same sets of licenses and exceptions.
func AreMatching(a, b string) bool {
	ta := internal.Tokenise(a)
	tb := internal.Tokenise(b)
	return internal.AreTokensListsMatchingSwap(ta, tb)
}

// License/exception text file
type File struct {
	Text          string
	SuggestedName string
}

// Return list of files for licenses/exceptions found in provided expression.
func GetFiles(expr string) (
	licenses map[string]File,
	exceptions map[string]File,
	unknown []string,
) {
	licenses = make(map[string]File)
	exceptions = make(map[string]File)
	unknown = make([]string, 0)

	tokens := internal.Tokenise(expr)
	for i := range len(tokens) {
		if slices.Contains(internal.Keywords, tokens[i]) {
			continue
		}
		if _, ok := internal.Globs[tokens[i]]; ok {
			tokens[i] = internal.GlobToFirstMatch(tokens[i])
		}
	}
	mapping := internal.TokensToShort(tokens)
	for _, token := range tokens {
		if slices.Contains(internal.Keywords, token) {
			continue
		}
		text := internal.GetText(token)
		if text == nil {
			unknown = append(unknown, token)
			continue
		}
		if slices.Contains(internal.ExceptionsList, token) {
			exceptions[token] = File{*text, mapping[token]}
			continue
		}
		licenses[token] = File{*text, mapping[token]}
	}
	return
}
