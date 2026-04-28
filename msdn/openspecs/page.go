package openspecs

// page.go contains the implementation of the Page and Section types, which represent the structure of
// documentation pages in the Microsoft Open Specifications, along with methods for parsing and extracting
// content from HTML documents using the goquery library. The Page type represents a documentation page, while
// the Section type represents a section within a documentation page, each containing its own name and
// documentation content.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// Page represents a documentation page in the Microsoft Open Specifications.
type Page struct {
	// UUID is the unique identifier for the page, used in URLs to access the page.
	UUID string `json:"uuid"`
	// Name is the name of the page, which is typically a human-readable title.
	Name string `json:"name"`
	// Documentation is a list of strings that provide the main content of the page, including
	// descriptions, explanations, and other relevant information.
	Documentation []string `json:"documentation"`
	// Sections is a list of sections within the page, where each section contains its
	// own name and documentation.
	Sections []*Section `json:"sections"`
	// Raw contains the raw HTML content of the page, which can be used for reference or further processing.
	Raw []byte `json:"raw"`
}

// GetSection retrieves a section from the page by its name. It returns the section and a boolean indicating
// whether the section was found. The method checks for both the exact name and a version of the name
// with a "__" prefix, which is a common convention in documentation for certain types of sections.
func (p *Page) GetSection(name string) (*Section, bool) {

	if p == nil {
		return nil, false
	}

	for _, section := range p.Sections {
		if section.Name == name || section.Name == strings.TrimPrefix(name, "__") {
			return section, true
		}
	}

	return nil, false
}

// GetObjectSection retrieves a section from the page by its name, specifically handling special cases for "This",
// "That", and "Return Values". It returns the section and a boolean indicating whether the section was found.
func (p *Page) GetObjectSection(name string) (*Section, bool) {

	switch name {
	case "This":
		return &Section{
			Name:          "This",
			Documentation: []string{"This: ORPCTHIS structure that is used to send ORPC extension data to the server."},
		}, true
	case "That":
		return &Section{
			Name:          "That",
			Documentation: []string{"That: ORPCTHAT structure that is used to return ORPC extension data to the client."},
		}, true
	case "ReturnValue", "Return Values":
		name = "Return Values"
	}

	return p.GetSection(name)
}

// Merge combines the content of the current page with another page, merging their documentation and sections.
func (p *Page) Merge(other *Page) *Page {

	if p == nil {
		return other
	}

	if other == nil {
		return p
	}

	merged := &Page{
		Name: p.Name,
		UUID: p.UUID,
	}

	merged.Documentation = append(merged.Documentation, p.Documentation...)
	if len(p.Documentation) == 0 || len(other.Documentation) == 0 ||
		p.Documentation[0] != other.Documentation[0] {
		merged.Documentation = append(merged.Documentation, other.Documentation...)
	}

	lookup := make(map[string]*Section)

	for _, sections := range [][]*Section{p.Sections, other.Sections} {
		for _, section := range sections {
			s, ok := lookup[section.Name]
			if !ok {
				s = &Section{Name: section.Name}
				merged.Sections, lookup[section.Name] = append(merged.Sections, s), s
			}

			if len(s.Documentation) == 0 || len(section.Documentation) == 0 ||
				s.Documentation[0] != section.Documentation[0] {
				s.Documentation = append(s.Documentation, section.Documentation...)
			}
		}
	}

	return merged
}

func (p *Page) Unmarshal(ctx context.Context, r io.Reader) error {

	document, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return fmt.Errorf("creating document from reader: %w", err)
	}

	if _, err := p.FromDocument(ctx, document); err != nil {
		return fmt.Errorf("parsing document: %w", err)
	}

	if p.Raw, err = PageDocumentToRaw(ctx, p.Name, p.UUID, document); err != nil {
		return fmt.Errorf("rendering raw html: %w", err)
	}

	return nil
}

func (p *Page) Marshal(ctx context.Context) (io.Reader, error) {
	if p.Raw == nil {
		return nil, fmt.Errorf("raw content is nil")
	}
	return bytes.NewReader(p.Raw), nil
}

// AddSection adds a new section with the specified name and documentation to the page.
func (p *Page) AddSection(section string, doc string) *Page {
	s := &Section{Name: section}
	p.Sections = append(p.Sections, s.AddDocumentation(doc))
	return p
}

// skipDocs is a set of documentation strings that should be ignored when adding
// documentation to a page or section.
var skipDocs = map[string]struct{}{
	"":                        {},
	"msdn link":               {},
	"Server Processing Rules": {},
}

// AddDocumentation adds a documentation string to the page.
func (p *Page) AddDocumentation(doc string) *Page {
	if _, ok := skipDocs[doc]; ok {
		return p
	}

	if len(p.Sections) > 0 {
		p.Sections[len(p.Sections)-1].AddDocumentation(doc)
		return p
	}

	p.Documentation = append(p.Documentation, doc)
	return p
}

// Section represents a section within a documentation page, containing a name and its own documentation.
type Section struct {
	// Name is the name of the section, which serves as a heading for the content within that section.
	Name string `json:"name"`
	// Documentation is a list of strings that provide the content of the section, including descriptions,
	// explanations, and other relevant information specific to that section.
	Documentation []string `json:"documentation"`
}

// Lines returns the documentation of the section as a slice of strings, where each string
// represents a line of text.
func (p *Section) Lines(size int) []string {

	s := &strings.Builder{}

	P := func(a ...any) {
		fmt.Fprintln(s, a...)
	}

	for _, doc := range p.Documentation {
		renderLine(P, doc, size)
	}

	return strings.Split(s.String(), "\n")
}

// AddDocumentation adds a documentation string to the section.
func (s *Section) AddDocumentation(doc string) *Section {
	if doc != "" {
		s.Documentation = append(s.Documentation, doc)
	}
	return s
}

var (
	nameRe = regexp.MustCompile(`([A-Za-z0-9_ \xa0]+)(?:<\d+>)?(?:\((?:(?:variable)|(?:\d+ (?:bytes?|words?)))\))?\s*:`)
)

// parseName attempts to extract a section name from the provided string using a regular expression.
// It returns the extracted name and a boolean indicating whether the extraction was successful.
func parseName(n string) (string, bool) {
	m := nameRe.FindStringSubmatch(n)
	if len(m) > 1 {
		return strings.TrimSuffix(strings.TrimSpace(m[1]), ":"), true
	}
	return n, false
}

// FromDocument populates the Page struct by parsing the provided goquery.Document, extracting relevant
// information such as section names and documentation content, and storing the raw HTML for reference.
func (p *Page) FromDocument(ctx context.Context, document *goquery.Document) (*Page, error) {

	if val, ok := document.Attr("name"); ok && val != "" {
		p.Name = val
	}

	if val, ok := document.Attr("uuid"); ok && val != "" {
		p.UUID = val
	}

	document.Find("div.content > :not(h1, div, nav)").Each(func(_ int, child *goquery.Selection) {
		switch goquery.NodeName(child) {
		case "p":
			// field description or struct description.
			found := false
			child.Find("b").Each(func(_ int, b *goquery.Selection) {
				if name, ok := parseName(DocString(b)); ok {
					found = true
					p.AddSection(name, "")
				}
			})

			if !found {
				child.Find("i").Each(func(_ int, i *goquery.Selection) {
					if name, ok := parseName(DocString(i)); ok {
						found = true
						p.AddSection(name, "")
					}
				})
			}

			txt := DocString(child)
			if !found {

				for _, prefix := range DefaultPrefixes {
					if prefix.Match(txt) {
						p.AddSection(prefix.Name, "")
						break
					}
				}
			}

			p.AddDocumentation(txt)

		case "dl":

			l := ""

			var f func(*html.Node)
			f = func(n *html.Node) {
				if n.Type == html.TextNode {
					l += n.Data
				} else if n.Type == html.ElementNode {
					qn := &goquery.Selection{Nodes: []*html.Node{n}}
					switch n.Data {
					case "pre":
						p.AddDocumentation("<pre>\n" + strings.TrimSpace(qn.Text()) + "\n</pre>")
						return
					case "p":

						found := false
						qn.Find("b").Each(func(_ int, b *goquery.Selection) {
							if name, ok := parseName(DocString(b)); ok {
								found = true
								p.AddSection(name, "")
							}
						})

						if found {
							p.AddDocumentation(DocString(qn))
							return
						}

					case "dl", "dt", "dd", "ul", "ol":
						p.AddDocumentation(DocString(texter(l)))
						l = ""
					case "table":
						p.AddDocumentation(DocString(texter(l)))
						l = ""
						p.AddDocumentation(RenderTable(ctx, &goquery.Selection{
							Nodes: []*html.Node{n},
						}))
						return
					case "img":
						p.AddDocumentation(DocString(texter(l)))
						l = ""
						p.AddDocumentation(RenderImage(ctx, &goquery.Selection{
							Nodes: []*html.Node{n},
						}))
						return
					case "li":
						strL := DocString(texter(l))
						if strL != "" {
							p.AddDocumentation("  *  " + DocString(texter(l)))
						}

						l = ""
					}
				}

				if n.FirstChild != nil {
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						f(c)
					}
				}
			}

			for _, n := range child.Nodes {
				f(n)
			}

			if l != "" {
				p.AddDocumentation(DocString(texter(l)))
			}

		case "ul":
			txt, add := RenderHTML(ctx, child), false
			for _, line := range strings.Split(txt, "\n") {
				if line = strings.TrimSpace(line); line == "" {
					continue
				}
				if strings.Count(line, "*") == 1 {
					add = true
					continue
				}

				if add {
					add, line = false, "*"+" "+line
					p.AddDocumentation(line)
				} else {
					p.AddDocumentation(line)
				}
			}

		case "ol":
			p.AddDocumentation(RenderHTML(ctx, child))
		case "table":
			p.AddDocumentation(RenderTable(ctx, child))
		}
	})

	var err error

	// store the raw HTML content of the page for reference or further processing.
	if p.Raw, err = PageDocumentToRaw(ctx, p.Name, p.UUID, document); err != nil {
		return nil, fmt.Errorf("rendering raw html: %w", err)
	}

	return p, nil
}

// PageDocumentToRaw extracts the raw HTML content from the provided goquery.Document,
// specifically targeting the main content of the page while excluding certain elements
// like headers, divs, and navigation.
func PageDocumentToRaw(ctx context.Context, name, uuid string, document *goquery.Document) ([]byte, error) {

	w, nodes := bytes.NewBuffer(nil), document.Find("main div.content > :not(h1, div, nav)").Nodes

	if document.Find("main").Length() == 0 {
		nodes = document.Find("div.content > :not(h1, div, nav)").Nodes
	}

	if _, err := w.Write([]byte(fmt.Sprintf(`<div class="content" name="%s" uuid="%s">`, name, uuid))); err != nil {
		return nil, fmt.Errorf("writing main content start: %w", err)
	}

	if len(nodes) == 0 {
		w.WriteString("<debug>no-nodes-found</debug>")
	}

	for _, n := range nodes {
		if err := html.Render(w, n); err != nil {
			return nil, fmt.Errorf("rendering node: %w", err)
		}
	}

	if _, err := w.Write([]byte(`</div>`)); err != nil {
		return nil, fmt.Errorf("writing main content end: %w", err)
	}

	return w.Bytes(), nil
}

// Render generates a string representation of the page, including its name, documentation,
// and sections, formatted in a way that is suitable for display or further processing, with
// line breaks and indentation for better readability.
func (p *Page) Render() string {

	s := &strings.Builder{}

	P := func(a ...any) {
		fmt.Fprintln(s, append([]any{"//"}, a...)...)
	}

	P("#", p.Name)

	for _, doc := range p.Documentation {
		P()
		renderLine(P, doc, 80)
	}

	for _, section := range p.Sections {
		P()
		P("##", section.Name)
		for _, doc := range section.Documentation {
			P()
			renderLine(P, doc, 80)
		}
	}

	return s.String()
}

// Lines returns the documentation of the page as a slice of strings, where each string
// represents a line of text, formatted with line breaks and indentation for better readability.
func (p *Page) Lines(size int) []string {

	s := &strings.Builder{}

	P := func(a ...any) {
		fmt.Fprintln(s, a...)
	}

	for _, doc := range p.Documentation {
		renderLine(P, doc, size)
	}

	return strings.Split(s.String(), "\n")
}

func renderLine(P func(...any), doc string, width int) {
	tab := false
	if strings.Contains(doc, "|") || strings.Contains(doc, "+---") || strings.Contains(doc, "<pre>") {
		tab = true
	}

	doc = strings.ReplaceAll(doc, "<pre>", "")
	doc = strings.ReplaceAll(doc, "</pre>", "")

	for _, doc := range strings.Split(doc, "\n") {
		doc = strings.ReplaceAll(doc, "… ", "... ")
		if strings.Contains(doc, "|") || strings.Contains(doc, "+---") || tab {
			P("\t", doc)
			continue
		}
		line := ""
		for _, word := range strings.Split(doc, " ") {
			line += word + " "
			if len(line) > width {
				P(line)
				line = ""
			}
		}
		if len(line) > 0 {
			P(line)
		}
	}
}
