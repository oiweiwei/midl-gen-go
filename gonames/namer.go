// Package gonames converts IDL identifiers to Go names.
//
// A name is produced in stages (see Namer.Words): the whole identifier may be
// renamed, prefixes and suffixes are trimmed, the rest is split into lexems
// (Split) which the rules may split further or join, each lexem is dropped,
// abbreviated or capitalized, and a few words are moved or dropped by position.
// The rules come from YAML files, see Config and README.md.
package gonames

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Title capitalizes the first letter of the word and lower cases the rest.
var Title = cases.Title(language.English).String

// Escape marks the identifier as a Go name that must not be converted.
func Escape(n string) string { return "%" + n }

// Unescape returns the identifier without the Escape mark.
func Unescape(n string) string { return strings.TrimPrefix(n, "%") }

// Namer applies a set of rules to IDL identifiers. It is immutable and safe for
// concurrent use. A nil *Namer applies the built-in rules.
type Namer struct {
	cfg     *Config
	prefix  map[string]struct{}
	word    map[string]struct{}
	acronym map[string]string // lower case -> spelling.
	maxJoin int               // the most lexems an acronym is made of.
}

// NewNamer returns the namer for the rules; nil rules are no rules.
func NewNamer(cfg *Config) *Namer {
	if cfg == nil {
		cfg = &Config{}
	}
	nr := &Namer{cfg: cfg, prefix: map[string]struct{}{}, word: map[string]struct{}{}, acronym: map[string]string{}}
	for _, p := range cfg.Trim.Prefixes {
		nr.prefix[p] = struct{}{}
	}
	for _, w := range cfg.Trim.Words {
		nr.word[w] = struct{}{}
	}
	for _, a := range cfg.Acronyms {
		nr.acronym[strings.ToLower(a)] = a
	}
	return nr
}

// Config returns the rules of the namer; nil for a nil namer.
func (nr *Namer) Config() *Config {
	if nr == nil {
		return nil
	}
	return nr.cfg
}

// Resolve returns the namer, or the built-in one for nil, so that code that
// has no per-IDL rules can pass nil.
func Resolve(nr *Namer) *Namer {
	if nr == nil {
		return defaultNamer
	}
	return nr
}

// GoName converts the IDL identifier to a Go name.
func GoName(nr *Namer, n string) string {
	return strings.Join(Resolve(nr).Words(n, true), "")
}

// GoNamePrivate converts the IDL identifier to an unexported Go name.
func GoNamePrivate(nr *Namer, n string) string {
	words := Resolve(nr).Words(n, true)
	if len(words) > 0 {
		words[0] = strings.ToLower(words[0])
	}
	ret := strings.Join(words, "")
	if ret == "import" {
		ret += "1"
	}
	return ret
}

// GoMemberName is GoName for a member of the owner, see Namer.MemberWords.
func GoMemberName(nr *Namer, owner, name string) string {
	return strings.Join(Resolve(nr).MemberWords(owner, name, true), "")
}

// GoMemberNameNoReserved is GoMemberName without the replacement of reserved fields.
func GoMemberNameNoReserved(nr *Namer, owner, name string) string {
	return strings.Join(Resolve(nr).MemberWords(owner, name, false), "")
}

// GoNameNoReserved is GoName without the replacement of reserved fields.
func GoNameNoReserved(nr *Namer, n string) string {
	return strings.Join(Resolve(nr).Words(n, false), "")
}

// GoSnakeCase converts the IDL identifier to a lower case snake case name.
func GoSnakeCase(nr *Namer, n string) string {
	words := Resolve(nr).Words(n, true)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, "_")
}

// GoLexName returns the words of the Go name of the IDL identifier.
func GoLexName(nr *Namer, n string) []string { return Resolve(nr).Words(n, true) }

// GoMergeNames joins two names, dropping the part of the second that repeats the first.
func GoMergeNames(nr *Namer, n1, n2 string) string { return Resolve(nr).MergeNames(n1, n2) }

// reserved matches the names of fields that stand for padding or are unused.
var reserved = regexp.MustCompile(`^(Word)?(Reserved?|Unused?|Dummy)(String|Long|Array|MustBeZero|Field|Pad)?(\d+)?(Type|Size)?$`)

// Words returns the words of the Go name of the IDL identifier. If reserved is
// true a reserved field (padding, unused) gets the blank name "_".
func (nr *Namer) Words(name string, reserved bool) []string {
	return Resolve(nr).words(name, reserved, nil)
}

// MemberWords is Words for a member (a field, parameter or element) of the
// owner (a struct, union, enum or operation): a rule written for exactly that
// member in the override section applies instead of the general rules.
func (nr *Namer) MemberWords(owner, name string, reserved bool) []string {
	return Resolve(nr).member(owner, name, reserved, nil)
}

// ExplainMember is MemberWords that also returns how the name was derived.
func (nr *Namer) ExplainMember(owner, name string, reserved bool) *Trace {
	tr := &Trace{Input: name}
	tr.Words = Resolve(nr).member(owner, name, reserved, tr)
	return tr
}

func (nr *Namer) member(owner, name string, reserved bool, tr *Trace) []string {
	if words, ok := nr.cfg.Override[owner][name]; ok {
		tr.add(RuleOverride, "%s.%s -> %v", owner, name, words)
		return append([]string(nil), words...)
	}
	return nr.words(name, reserved, tr)
}

// Explain is Words that also returns how the name was derived.
func (nr *Namer) Explain(name string, reserved bool) *Trace {
	tr := &Trace{Input: name}
	tr.Words = Resolve(nr).words(name, reserved, tr)
	return tr
}

func (nr *Namer) words(name string, reserved bool, tr *Trace) []string {

	if name == "" {
		return nil
	}

	// a name that is escaped is taken as it is.
	if lit := Unescape(name); lit != name {
		tr.add(RuleEscape, "%s taken as it is", lit)
		return []string{lit}
	}

	if words, ok := nr.cfg.Rename[name]; ok {
		tr.add(RuleRename, "%s -> %v", name, words)
		return append([]string(nil), words...)
	}

	lexems := nr.lex(nr.trim(name, tr), tr)

	if tr != nil {
		tr.Lexems = append([]string(nil), lexems...)
	}

	words, moved := nr.spellAll(lexems, tr)

	if reserved && isReserved(words) {
		tr.add(RuleReserved, "%v is a reserved field", words)
		return []string{"_"}
	}

	words = placeMoved(words, moved, tr)
	words = nr.lastResort(words, lexems, tr)

	return dropContextHandle(words, tr)
}

// trim removes the prefixes and suffixes of the identifier (before it is split).
func (nr *Namer) trim(name string, tr *Trace) string {
	for _, prefix := range nr.cfg.Trim.PrefixAll {
		before := name
		// an identifier that is a prefix by itself is kept.
		if name = strings.TrimPrefix(name, prefix); name == "" {
			name = prefix
		}
		if name != before {
			tr.add(RulePrefixAll, "%s: %s -> %s", prefix, before, name)
		}
	}
	for _, suffix := range nr.cfg.Trim.Suffix {
		before := name
		name = strings.TrimSuffix(name, suffix)
		if name != before {
			tr.add(RuleSuffix, "%s: %s -> %s", suffix, before, name)
		}
	}
	return name
}

// lex splits the identifier into lexems (see Split) and applies the lookBehind
// and split rules to each of them in turn.
func (nr *Namer) lex(name string, tr *Trace) []string {
	var out []string
	for _, lexem := range Split(name) {
		var cur []rune
		out, cur = nr.lookBehind(out, []rune(lexem), tr)
		out = append(out, nr.split(cur, tr)...)
	}
	return out
}

// lookBehind applies the rule keyed by the previous and the current lexem: both
// are replaced by the words of the rule, the last of which stays the current
// lexem.
func (nr *Namer) lookBehind(out []string, cur []rune, tr *Trace) ([]string, []rune) {

	if len(out) < 1 {
		return out, cur
	}

	rule, ok := nr.cfg.LookBehind[strings.ToLower(out[len(out)-1])]
	if !ok {
		return out, cur
	}

	repl, ok := rule[strings.ToLower(string(cur))]
	if !ok {
		return out, cur
	}

	tr.add(RuleLookBehind, "%s + %s -> %v", out[len(out)-1], string(cur), repl)

	switch len(repl) {
	case 0:
		return out[:len(out)-1], nil
	case 1:
		return out[:len(out)-1], []rune(repl[0])
	}

	return append(out[:len(out)-1], repl[:len(repl)-1]...), []rune(repl[len(repl)-1])
}

// split returns the words of the lexem if there is a rule to split it.
func (nr *Namer) split(cur []rune, tr *Trace) []string {
	if words, ok := nr.cfg.Split[strings.ToLower(string(cur))]; ok {
		tr.add(RuleSplit, "%s -> %v", string(cur), words)
		return words
	}
	return []string{string(cur)}
}

// words turns each lexem into a word: it is dropped (trimmed prefix or word),
// abbreviated or capitalized. A first lexem with a rotate rule is not a word
// but is returned to be placed at the end.
func (nr *Namer) spellAll(lexems []string, tr *Trace) (words []string, moved string) {

	for i, lexem := range lexems {

		if lexem == "" {
			continue
		}

		lower := strings.ToLower(lexem)

		if i == 0 {
			if to, ok := nr.cfg.Rotate[lexem]; ok {
				tr.add(RuleRotate, "%s moves to the end as %s", lexem, to)
				moved = to
				continue
			}
			if _, ok := nr.prefix[lexem]; ok {
				tr.add(RuleTrimPrefix, "%s dropped", lexem)
				continue
			}
		}

		if _, ok := nr.word[lower]; ok {
			tr.add(RuleTrimWord, "%s dropped", lexem)
			continue
		}

		words = append(words, nr.spell(lower, tr))
	}

	return words, moved
}

func isReserved(words []string) bool { return reserved.MatchString(strings.Join(words, "")) }

// placeMoved puts the word of a rotate rule at the end of the name (rotate
// ob: Offset makes ObSize -> Size, Offset -> SizeOffset). It replaces the last
// Size or Length word if the moved word is one too, or the last word that
// ends with the beginning of the moved word (at least three letters), instead
// of being added.
func placeMoved(words []string, moved string, tr *Trace) []string {

	if moved == "" {
		return words
	}

loop:
	for j := len(words) - 1; j >= 0; j-- {
		if sizeLength(words[j]) && sizeLength(moved) {
			words[j], moved = moved, ""
			break
		}
		for i := len(moved); i >= 3 && len(words) > 0; i-- {
			if strings.HasSuffix(words[j], moved[:i]) {
				words[j], moved = moved, ""
				break loop
			}
		}
	}

	if moved != "" {
		words = append(words, moved)
	}

	tr.add(RulePlaceMoved, "%v", words)

	return words
}

// lastResort names an identifier that has no words left (everything was
// dropped) after its last lexem.
func (nr *Namer) lastResort(words, lexems []string, tr *Trace) []string {

	if len(words) > 0 || len(lexems) == 0 {
		return words
	}

	tr.add(RuleLastResort, "no words left, using %s", lexems[len(lexems)-1])

	return append(words, nr.spell(strings.ToLower(lexems[len(lexems)-1]), tr))
}

// spell returns the spelling of a lower case word in a name: the abbreviation
// of the word, the acronym, or the word capitalized.
func (nr *Namer) spell(lower string, tr *Trace) string {
	if abbr, ok := nr.cfg.Abbr[lower]; ok {
		tr.add(RuleAbbr, "%s -> %s", lower, abbr)
		return abbr
	}
	if acronym, ok := nr.acronym[lower]; ok {
		tr.add(RuleAcronym, "%s -> %s", lower, acronym)
		return acronym
	}
	return Title(lower)
}

// dropContextHandle removes the words "Context Handle" from a name of more than
// two words.
//
// Note: the words after a removal are not looked at again, so a second
// "Context Handle" pair directly behind the first stays.
func dropContextHandle(words []string, tr *Trace) []string {

	if len(words) <= 2 {
		return words
	}

	for i := 0; i < len(words)-1; i++ {
		if words[i] == "Context" && words[i+1] == "Handle" {
			tr.add(RuleContextHandle, "Context Handle dropped")
			words = append(words[:i], words[i+2:]...)
		}
	}

	return words
}

func sizeLength(s string) bool {
	s = strings.ToLower(s)
	return s == "size" || s == "length"
}

// MergeNames joins two names into one, dropping from the second name the words
// that repeat the end of the first one.
func (nr *Namer) MergeNames(n1, n2 string) string {

	nr = Resolve(nr)

	l1, l2 := nr.Words(n1, true), nr.Words(n2, false)

	i, j := 0, 0
	for i, j = 0, 0; j < len(l1) && i < len(l2); i, j = i+1, j+1 {

		p1, p2 := strings.ToLower(l1[j]), strings.ToLower(l2[i])
		if p1 == p2 {
			continue
		}

		if len(p1) > 2 && len(p2) > 2 && (strings.HasPrefix(p1, p2) || strings.HasPrefix(p2, p1)) {
			continue
		}

		break
	}

	if len(l2[i:]) == 0 {
		i--
	}

	for len(l1) > 0 && i < len(l2) && l1[len(l1)-1] == l2[i] {
		i++
	}

	acronym := ""
	for _, lex := range l1 {
		acronym += strings.ToLower(lex[:1])
	}

	if i < len(l2) && strings.ToLower(l2[i]) == acronym {
		i++
	}

	// typeflagtype
	if len(l1) > 1 && i+1 < len(l2) && strings.ToLower(l2[i]) == strings.ToLower(l1[len(l1)-2]) {
		i++
		for i < len(l2) && l2[i] == l1[len(l1)-1] {
			i++
		}
	}

	return strings.Join(nr.Words(strings.Join(append(l1, l2[i:]...), "_"), true), "")
}
