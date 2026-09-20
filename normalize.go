package guard

import (
	"encoding/base64"
	"regexp"
	"strings"
	"unicode"
)

// Normalizer turns one input string into candidate plaintexts. Every candidate
// is concatenated onto the text handed to the classifier, so a payload hidden
// in an encoding the model cannot internally invert is surfaced as plaintext
// before the model ever sees it.
//
// This is the fix for the measured miss: a bare ROT13 payload scored 0.32
// (PASS) because the classifier does not apply ROT13 — but it reads decoded
// plaintext perfectly. Decode here; judge there.
type Normalizer interface {
	Name() string
	Expand(in string) ([]string, bool)
}

// NormalizerFunc adapts a function to the Normalizer interface.
type NormalizerFunc struct {
	Label string
	Fn    func(string) ([]string, bool)
}

func (n NormalizerFunc) Name() string                      { return n.Label }
func (n NormalizerFunc) Expand(in string) ([]string, bool) { return n.Fn(in) }

// DefaultNormalizers is the stdlib-only pre-classification chain.
func DefaultNormalizers() []Normalizer {
	return []Normalizer{
		NormalizerFunc{"base64", expandBase64},
		NormalizerFunc{"hex", expandHex},
		NormalizerFunc{"rot13", expandRot13},
		NormalizerFunc{"url", expandURL},
		NormalizerFunc{"invisible-strip", expandStripInvisible},
		NormalizerFunc{"homoglyph-fold", expandFoldHomoglyphs},
	}
}

// maxExpanded bounds how much decoded text we append, so a hostile input cannot
// blow up the classifier's token bill.
const maxExpanded = 20000

// NormalizeAll runs the whole chain and returns the text to classify plus the
// names of the transforms that actually produced something.
func NormalizeAll(in string, chain []Normalizer) (string, []string) {
	if chain == nil {
		chain = DefaultNormalizers()
	}
	var parts []string
	var applied []string

	cur := in
	for _, n := range chain {
		outs, ok := n.Expand(cur)
		if !ok || len(outs) == 0 {
			continue
		}
		applied = append(applied, n.Name())
		for _, o := range outs {
			if o == "" || o == cur {
				continue
			}
			parts = append(parts, o)
			cur = o // chain: decode-of-decode is a real evasion shape
		}
	}

	if len(parts) == 0 {
		// Nothing decoded. Still hand over the invisible-stripped/folded form,
		// which changes nothing semantically but removes zero-width games.
		if s := stripInvisible(in); s != in {
			return in + "\n---\n" + s, append(applied, "invisible-strip")
		}
		return in, applied
	}

	joined := in + "\n---\n" + strings.Join(parts, "\n---\n")
	if len(joined) > maxExpanded {
		joined = joined[:maxExpanded]
	}
	return joined, applied
}

var b64Token = regexp.MustCompile(`[A-Za-z0-9+/]{16,}={0,2}`)

func expandBase64(in string) ([]string, bool) {
	var out []string
	for _, tok := range b64Token.FindAllString(in, 8) {
		if dec, err := base64.StdEncoding.DecodeString(padB64(tok)); err == nil {
			if printable(dec) {
				out = append(out, string(dec))
			}
		}
		if dec, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(tok, "=")); err == nil {
			if printable(dec) {
				out = append(out, string(dec))
			}
		}
	}
	return out, len(out) > 0
}

func padB64(s string) string {
	if m := len(s) % 4; m != 0 {
		return s + strings.Repeat("=", 4-m)
	}
	return s
}

var hexToken = regexp.MustCompile(`\b[0-9a-fA-F]{16,}\b`)

func expandHex(in string) ([]string, bool) {
	var out []string
	for _, tok := range hexToken.FindAllString(in, 8) {
		if len(tok)%2 != 0 {
			continue
		}
		b := make([]byte, len(tok)/2)
		ok := true
		for i := 0; i < len(tok); i += 2 {
			v, err := hexVal(tok[i])<<4|hexVal(tok[i+1]), error(nil)
			if err != nil {
				ok = false
				break
			}
			b[i/2] = byte(v)
		}
		if ok && printable(b) {
			out = append(out, string(b))
		}
	}
	return out, len(out) > 0
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

// rot13 is applied unconditionally: it is cheap, reversible, and it is the
// exact transform that defeated the unfixed guard.
func expandRot13(in string) ([]string, bool) {
	out := make([]rune, 0, len(in))
	changed := false
	for _, r := range in {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, 'a'+(r-'a'+13)%26)
			changed = true
		case r >= 'A' && r <= 'Z':
			out = append(out, 'A'+(r-'A'+13)%26)
			changed = true
		default:
			out = append(out, r)
		}
	}
	if !changed {
		return nil, false
	}
	return []string{string(out)}, true
}

var urlEsc = regexp.MustCompile(`%[0-9a-fA-F]{2}`)

func expandURL(in string) ([]string, bool) {
	if !urlEsc.MatchString(in) {
		return nil, false
	}
	var b strings.Builder
	for i := 0; i < len(in); i++ {
		if in[i] == '%' && i+2 < len(in) {
			hi, lo := hexVal(in[i+1]), hexVal(in[i+2])
			b.WriteByte(byte(hi<<4 | lo))
			i += 2
			continue
		}
		b.WriteByte(in[i])
	}
	return []string{b.String()}, true
}

func stripInvisible(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u00ad':
			return -1
		}
		if unicode.Is(unicode.Cf, r) { // format chars
			return -1
		}
		return r
	}, s)
}

func expandStripInvisible(in string) ([]string, bool) {
	out := stripInvisible(in)
	if out == in {
		return nil, false
	}
	return []string{out}, true
}

// homoglyphs maps the common non-Latin lookalikes to their Latin twin, so
// "ignоre" with a Cyrillic о is not a different word to the classifier.
var homoglyphs = map[rune]rune{
	'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'у': 'y', 'х': 'x',
	'і': 'i', 'ј': 'j', 'ѕ': 's', 'ԁ': 'd', 'ɡ': 'g', 'ⅰ': 'i',
	'Α': 'A', 'Β': 'B', 'Ε': 'E', 'Ζ': 'Z', 'Η': 'H', 'Ι': 'I', 'Κ': 'K',
	'Μ': 'M', 'Ν': 'N', 'Ο': 'O', 'Ρ': 'P', 'Τ': 'T', 'Υ': 'Y', 'Χ': 'X',
	'α': 'a', 'ε': 'e', 'ι': 'i', 'κ': 'k', 'ο': 'o', 'ρ': 'p', 'τ': 't', 'υ': 'u', 'χ': 'x',
	'０': '0', '１': '1', '２': '2', '３': '3', '４': '4', '５': '5',
	'６': '6', '７': '7', '８': '8', '９': '9',
}

func expandFoldHomoglyphs(in string) ([]string, bool) {
	changed := false
	out := strings.Map(func(r rune) rune {
		if lat, ok := homoglyphs[r]; ok {
			changed = true
			return lat
		}
		return r
	}, in)
	if !changed {
		return nil, false
	}
	return []string{out}, true
}

func printable(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	good := 0
	for _, c := range b {
		if c == '\n' || c == '\t' || (c >= 32 && c < 127) {
			good++
		}
	}
	return good*100/len(b) >= 85
}
