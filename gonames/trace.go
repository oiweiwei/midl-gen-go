package gonames

import "fmt"

// Rule names of the steps of a Trace.
const (
	RuleOverride      = "override"
	RuleEscape        = "escape"
	RuleRename        = "rename"
	RulePrefixAll     = "trim.prefixAll"
	RuleSuffix        = "trim.suffix"
	RuleLookBehind    = "lookBehind"
	RuleSplit         = "split"
	RuleRotate        = "rotate"
	RuleTrimPrefix    = "trim.prefix"
	RuleTrimWord      = "trim.word"
	RuleAbbr          = "abbr"
	RuleAcronym       = "acronym"
	RuleReserved      = "reserved"
	RulePlaceMoved    = "rotate.place"
	RuleLastResort    = "lastResort"
	RuleContextHandle = "contextHandle"
)

// Step is one thing that happened to the name.
type Step struct {
	// Rule is the rule that fired, one of the Rule constants.
	Rule string
	// Detail says what it did, e.g. "acl -> ACL".
	Detail string
}

// Trace records how a name was derived: the rules that fired, in order, and
// the words the name ended up as. See Namer.Explain.
type Trace struct {
	// Input is the IDL identifier.
	Input string
	// Lexems are the pieces the identifier was split into, before the rules for
	// words were applied.
	Lexems []string
	// Steps are the rules that fired.
	Steps []Step
	// Words are the words of the Go name.
	Words []string
}

// add records a step; it does nothing on a nil Trace, so the pipeline runs
// the same with and without tracing.
func (t *Trace) add(rule, format string, args ...any) {
	if t == nil {
		return
	}
	t.Steps = append(t.Steps, Step{Rule: rule, Detail: fmt.Sprintf(format, args...)})
}

// Rules returns the distinct rules that fired, in order of first use.
func (t *Trace) Rules() []string {
	var rules []string
	seen := map[string]bool{}
	for _, s := range t.Steps {
		if !seen[s.Rule] {
			seen[s.Rule] = true
			rules = append(rules, s.Rule)
		}
	}
	return rules
}
