package openspecs

// client.go contains the implementation of the Client, which is responsible for fetching and parsing
// the documentation pages from the Microsoft Open Specifications.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog"
)

type Client struct {
	Indexer *ProtocolIndexer
	Cache   *CacheFS
}

// GetProtocolDocumentationURL constructs the URL for the documentation of a specific protocol
// in the Microsoft Open Specifications.
func (c *Client) GetProtocolDocumentationURL(protocolName string) string {

	protocolName = strings.ToLower(protocolName)

	if !strings.Contains(protocolName, "/") {
		if !strings.Contains(protocolName, "-") {
			protocolName = msPrefix + protocolName
		}
		protocolName = WindowsProtocols + "/" + protocolName
	}

	return MustJoinURL(OpenSpecsBaseURL, protocolName)
}

// GetProtocolIndexURL constructs the URL for the index of a specific protocol in the Microsoft Open Specifications.
func (c *Client) GetPageURL(protocolName, pageUUID string) string {
	return MustJoinURL(
		c.GetProtocolDocumentationURL(protocolName),
		pageUUID,
	)
}

// MustGetDocument retrieves the HTML document from the specified URL, retrying indefinitely until it succeeds.
func (c *Client) MustGetDocument(ctx context.Context, url string) (*goquery.Document, error) {

	log := zerolog.Ctx(ctx).With().
		Str("url", url).
		Logger()

	ctx = log.WithContext(ctx)

	// do whatever it takes to get the document.
	for {
		// be aggressive with timeouts, as the documentation is often slow to respond
		// and we don't want to block the entire process for a single request.
		cli := &http.Client{Timeout: 1 * time.Second}

		resp, err := cli.Get(url)
		if err != nil {
			log.Warn().Err(err).Msg("fetching document")
			continue
		}

		body, err := readAllAndClose(ctx, resp.Body)
		if err != nil {
			log.Warn().Err(err).Msg("reading document body")
			continue
		}

		// the documentation is often malformed, so we need to fix it before parsing.
		body = bytes.ReplaceAll(body, []byte(`</b>:`), []byte(`:</b>`))

		document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			log.Warn().Err(err).Msg("parsing document")
			continue
		}

		return document, nil
	}
}

func (c *Client) RelPath(ctx context.Context, protocolName, name string) string {

	log := zerolog.Ctx(ctx).With().Logger()

	if c.Cache == nil {
		log.Debug().Msg("no cache configured, skipping lookup")
		return ""
	}

	if c.Indexer == nil {
		log.Debug().Msg("no indexer configured, skipping lookup")
		return filepath.Join(WindowsProtocols, protocolName, name)
	}

	index, ok := c.Indexer.Get(protocolName)
	if !ok {
		log.Debug().Msg("protocol not indexed, skipping lookup")
		return ""
	}

	rel := filepath.Join(index.Family, index.Name, name)

	return rel
}

// Lookup attempts to retrieve the cached data for a specific protocol and name, and if it exists,
// unmarshals it into the provided Unmarshaler. It returns true if the data was found and unmarshaled
// successfully, false if it was not found, and an error if there was an issue reading from the cache
// or unmarshaling the data.
func (c *Client) Lookup(ctx context.Context, protocolName, name string, val Cacher) (bool, error) {

	log := zerolog.Ctx(ctx).With().Str("lookup", name).Logger()
	ctx = log.WithContext(ctx)

	rel := c.RelPath(ctx, protocolName, name)
	if rel == "" {
		return false, nil
	}

	log = log.With().Str("path", rel).Logger()
	ctx = log.WithContext(ctx)

	r, err := c.Cache.Read(rel)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Msg("cache miss")
			return false, nil
		}
		return false, err
	}

	log.Debug().Msg("cache hit, unmarshaling")

	if err := val.Unmarshal(ctx, r); err != nil {
		return false, fmt.Errorf("unmarshaling cached data: %w", err)
	}

	log.Debug().Msg("unmarshaled cached data successfully")

	return true, nil
}

func (c *Client) Flush(ctx context.Context, protocolName, name string, val Cacher) error {

	log := zerolog.Ctx(ctx).With().Str("flush", name).Logger()
	ctx = log.WithContext(ctx)

	rel := c.RelPath(ctx, protocolName, name)
	if rel == "" {
		return nil
	}

	log = log.With().Str("path", rel).Logger()
	ctx = log.WithContext(ctx)

	r, err := val.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("marshaling value for cache: %w", err)
	}

	if err := c.Cache.Write(rel, r); err != nil {
		return fmt.Errorf("writing to cache: %w", err)
	}

	log.Debug().Msg("flushed value to cache successfully")
	return nil
}

func (c *Client) GetProtocolIndex(ctx context.Context, protocolName string) (Index, error) {

	log := zerolog.Ctx(ctx).With().
		Str("component", "openspecs").
		Str("protocol", protocolName).
		Logger()

	ctx = log.WithContext(ctx)

	if c.Indexer == nil {
		return nil, fmt.Errorf("no indexer configured")
	}

	index, ok := c.Indexer.Get(protocolName)
	if !ok {
		return nil, fmt.Errorf("protocol not indexed")
	}

	return c.GetProtocolIndexByID(ctx, protocolName, index.UUID)
}

const indexFile = "index.json"

// GetProtocolIndex retrieves the index of a specific protocol from the Microsoft Open Specifications.
func (c *Client) GetProtocolIndexByID(ctx context.Context, protocolName, indexUUID string) (Index, error) {

	log := zerolog.Ctx(ctx).With().
		Str("component", "openspecs").
		Str("protocol", protocolName).
		Str("index_uuid", indexUUID).
		Logger()

	ctx = log.WithContext(ctx)

	index := make(Index)

	found, err := c.Lookup(ctx, protocolName, indexFile, &index)
	if err != nil {
		log.Warn().Err(err).Msg("looking up index in cache")
	}

	if found {

		if c.Indexer != nil {
			for _, extra := range c.Indexer.GetExtra(protocolName) {
				name, uuid := extra.Name+" "+"structure", extra.UUID
				index[name] = map[string]string{name: uuid, "_source": "extra"}
			}
		}

		return index, nil
	}

	document, err := c.MustGetDocument(ctx, c.GetPageURL(protocolName, indexUUID))
	if err != nil {
		return nil, err
	}

	document.Find("div p").Each(func(_ int, s *goquery.Selection) {

		entry := DocString(s)
		if entry == "" {
			return
		}

		s.Find("a").Each(func(_ int, s *goquery.Selection) {
			if href, _ := s.Attr("href"); href != "" {
				index.Add(entry, DocString(s), href)
			}
		})

		if len(index[entry]) == 0 {
			delete(index, entry)
		}
	})

	if err := c.Flush(ctx, protocolName, indexFile, &index); err != nil {
		log.Warn().Err(err).Msg("flushing index to cache")
	}

	if c.Indexer != nil {
		for _, extra := range c.Indexer.GetExtra(protocolName) {
			name, uuid := extra.Name+" "+"structure", extra.UUID
			index[name] = map[string]string{name: uuid, "_source": "extra"}
		}
	}

	return index, nil
}

// GetProtocolDocumentationPage retrieves the documentation page for a specific protocol and
// page UUID from the Microsoft Open Specifications.
func (c *Client) GetProtocolDocumentationPage(ctx context.Context, protocolName, pageName, pageUUID string) (*Page, error) {

	log := zerolog.Ctx(ctx).With().
		Str("component", "openspecs").
		Str("protocol", protocolName).
		Str("page_name", pageName).
		Str("page_uuid", pageUUID).
		Logger()

	ctx = log.WithContext(ctx)

	if strings.Contains(pageUUID, "/") {
		// if the pageUUID contains a slash, it means that the protocol name is actually part of the page UUID,
		// so we need to split it and adjust the protocol name accordingly.
		last := strings.LastIndex(pageUUID, "/")
		protocolName, pageUUID = pageUUID[:last], pageUUID[last+1:]
	}

	page := &Page{
		Name: pageName,
		UUID: pageUUID,
	}

	found, err := c.Lookup(ctx, protocolName, pageUUID, page)
	if err != nil {
		log.Warn().Err(err).Msg("looking up page in cache")
	}

	if found {
		return page, nil
	}

	document, err := c.MustGetDocument(ctx, c.GetPageURL(protocolName, pageUUID))
	if err != nil {
		return nil, err
	}

	if _, err := page.FromDocument(ctx, document); err != nil {
		log.Warn().Err(err).Msg("parsing page")
	}

	if err := c.Flush(ctx, protocolName, pageUUID, page); err != nil {
		log.Warn().Err(err).Msg("flushing page to cache")
	}

	return page, nil
}

// MustJoinURL joins the base URL with the provided paths, ensuring
// that the resulting URL is properly formatted.
func MustJoinURL(base string, paths ...string) string {

	u, err := url.Parse(base)
	if err != nil {
		panic(err)
	}

	return u.JoinPath(paths...).String()
}

type Texter interface{ Text() string }

type texter string

func (t texter) Text() string { return string(t) }

// DocString takes a string and returns a cleaned-up version of it, where all lines
// are trimmed of leading and trailing whitespace,
func DocString(q Texter) string {
	v := q.Text()
	s := strings.Split(v, "\n")
	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}

	return strings.TrimSpace(strings.Join(s, " "))
}

// msPrefix is the prefix used for protocol names in Microsoft Open Specifications
// when they are not already prefixed.
const msPrefix = "ms-"

// readAllAndClose reads all data from the provided io.ReadCloser and ensures
// that it is closed afterwards.
func readAllAndClose(ctx context.Context, body io.ReadCloser) ([]byte, error) {
	defer func() {
		if err := body.Close(); err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("closing response body")
		}
	}()
	return io.ReadAll(body)
}
