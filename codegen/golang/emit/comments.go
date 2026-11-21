package emit

import "strings"

func (p *Emitter) CB(f func()) {
	p.inCB = true
	f()
	p.inCB = false
}

func (p *Emitter) GenCommentLine(ctx *Env, line string) {
	p.GenComment(ctx, []string{line})
}

func (p *Emitter) GenComment(ctx *Env, docs []string) {
	p.CB(func() {
		for _, doc := range docs {
			for _, ddoc := range strings.Split(doc, "\n") {
				ddoc = strings.ReplaceAll(ddoc, "… ", "... ")
				if strings.Contains(ddoc, "|") || strings.Contains(ddoc, "+---") {
					p.P("\t", ddoc)
					continue
				}
				line := ""
				for _, word := range strings.Split(ddoc, " ") {
					line += word + " "
					if len(line) > 80 {
						p.P(line)
						line = ""
					}
				}
				if len(line) > 0 {
					p.P(line)
				}
			}
			p.P()
		}
	})
}
