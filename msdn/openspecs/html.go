package openspecs

// html.go contains the implementation of functions for rendering HTML content from the Microsoft Open Specifications
// into plain text or Markdown format, using the goquery library for HTML parsing and the html2text library for
// converting HTML to text. It includes functions for rendering images and tables, as well as a helper type for
// logging goquery selections.

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
	"jaytaylor.com/html2text"
)

// RenderImage takes a goquery selection representing an HTML image element and converts it into a
// Markdown-style image link, using the alt text as the link text and the src attribute as the URL.
func RenderImage(ctx context.Context, q *goquery.Selection) string {
	alt, ok := q.Attr("alt")
	if !ok {
		return ""
	}
	src, ok := q.Attr("src")
	if !ok {
		return ""
	}
	return "[" + alt + "](" + src + ")"
}

// RenderTable takes a goquery selection representing an HTML table and converts it into a
// plain text representation.
func RenderHTML(ctx context.Context, q *goquery.Selection) string {

	log := zerolog.Ctx(ctx).With().
		Str("method", "render_html").
		Stringer("html", goqueryStringer{q}).
		Logger()

	ret, err := html2text.FromHTMLNode(q.Nodes[0])
	if err != nil {
		log.Warn().Err(err).Msg("rendering html")
	}
	return ret
}

// RenderTable renders an HTML table into a plain text representation, handling both regular
// tables and those with colspan attributes.
func RenderTable(ctx context.Context, table *goquery.Selection) string {

	log := zerolog.Ctx(ctx).With().
		Str("method", "render_table").
		Stringer("html", goqueryStringer{table}).
		Logger()

	table.Find("td").Each(func(_ int, tx *goquery.Selection) {
		// normalize text.
		tx.SetText(DocString(tx))
	})

	buf := bytes.NewBuffer(nil)

	html, err := goquery.OuterHtml(table)
	if err != nil {
		log.Warn().Err(err).Msg("rendering table")
	}

	opts := html2text.Options{
		PrettyTables:        true,
		OmitLinks:           true,
		PrettyTablesOptions: html2text.NewPrettyTablesOptions(),
	}

	opts.PrettyTablesOptions.RowLine = true
	opts.PrettyTablesOptions.ColWidth = 80

	if !strings.Contains(html, "colspan=") {
		html, err := html2text.FromString(html, opts)
		if err != nil {
			log.Warn().Err(err).Msg("rendering table")
		}
		fmt.Fprintln(buf, html)
		return buf.String()
	}

	// render table header separately.
	html, err = goquery.OuterHtml(table.Find("th").First().Parent())
	if err != nil {
		log.Warn().Err(err).Msg("rendering table header")
	}

	html = "<table>" + html + "</table>"

	header, err := html2text.FromString(html, opts)
	if err != nil {
		log.Warn().Err(err).Msg("rendering table header")
	}

	last := ""
	for _, line := range strings.Split(header, "\n") {
		last = line
		fmt.Fprintln(buf, line)
	}

	table.Find("tr").Each(func(i int, tr *goquery.Selection) {

		if i == 0 {
			return
		}

		tr.Find("td").Each(func(i int, td *goquery.Selection) {

			attr, ok := td.Attr("colspan")
			if !ok {
				return
			}

			colspan, err := strconv.Atoi(attr)
			if err != nil {
				log.Warn().Err(err).Str("attr", attr).Msg("parse colspan")
			}

			txt := DocString(td)
			repeat := (colspan-1)*4 + 1 - len(txt)
			if repeat <= 0 {
				repeat = 10
			}

			fmt.Fprintf(buf, "| %s %s", txt, strings.Repeat(" ", repeat))
		})

		fmt.Fprintf(buf, "|\n")
		fmt.Fprintln(buf, last)

	})

	return buf.String()
}

// goqueryStringer is a helper type that implements the Stringer interface for a goquery selection,
// allowing it to be easily included in log messages.
type goqueryStringer struct {
	q *goquery.Selection
}

// String returns the HTML representation of the goquery selection, or an error message if the
// HTML cannot be rendered.
func (q goqueryStringer) String() string {
	html, err := goquery.OuterHtml(q.q)
	if err != nil {
		return "<error: " + err.Error() + ">"
	}
	return html
}
