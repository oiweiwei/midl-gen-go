package gonames

import "unicode"

// Split breaks an identifier into lexems at underscores and at case changes:
//
//	NtlmSsp   -> Ntlm, Ssp
//	NTLMSSP   -> NTLMSSP       (a run of capitals is one lexem)
//	HTTPServer -> HTTP, Server (a capital followed by a lower case letter starts a lexem)
//	rpc_bind  -> rpc, bind
//
// No rules are applied: the result depends on the characters only. A lexem
// is never empty.
func Split(name string) []string {

	var (
		runes = []rune(name)
		cur   []rune
		out   []string
	)

	flush := func() {
		if len(cur) > 0 {
			out = append(out, string(cur))
			cur = nil
		}
	}

	for i, r := range runes {

		switch {
		case r == '_':
			// '_' is the gap between lexems.
			flush()
			continue
		case unicode.IsUpper(r):
			// a capital starts a new lexem if it is followed by a lower case letter
			// (HTTPServer: S) or follows one (NtlmSsp: S); a capital in a run of
			// capitals does not.
			if (i+1 < len(runes) && unicode.IsLower(runes[i+1])) || (i > 0 && unicode.IsLower(runes[i-1])) {
				flush()
			}
		}

		cur = append(cur, r)
	}

	flush()

	return out
}
