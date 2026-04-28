package openspecs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/rs/zerolog"
)

type MSDN struct {
	mu sync.RWMutex

	// CacheFS is the root directory for caching downloaded documentation pages and indexes.
	CacheFS string
	// ProtocolIndexer is the indexer used to look up protocol documentation pages and indexes,
	// which can be built from a local file or other sources.
	Indexer *ProtocolIndexer

	pages   []*Page
	lookup  map[string]*Page
	indexes map[string]Index
}

func (m *MSDN) Index(ctx context.Context, protocolName string) (Index, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.indexes == nil {
		return nil, false
	}

	protocol, ok := m.Indexer.Get(protocolName)
	if !ok {
		return nil, false
	}

	index, ok := m.indexes[protocol.Name]
	if !ok {
		return nil, false
	}

	return index, true
}

// GetPage retrieves the documentation page for the specified protocol name and page names.
func (m *MSDN) GetPage(ctx context.Context, pages ...string) (*Page, bool) {

	m.mu.RLock()
	defer m.mu.RUnlock()

	var out *Page

	for _, p := range m.pages {
		for _, pageName := range pages {
			if p.Name == pageName {
				out = out.Merge(p)
			}
		}
	}

	return out, out != nil
}

// Sync synchronizes the documentation pages and indexes for the specified protocol name.
// It retrieves the protocol index and documentation pages, and updates the internal state of the MSDN struct.
func (m *MSDN) Sync(ctx context.Context, protocolName string) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	var pages []*Page
	var errs []error

	protocol, ok := m.Indexer.Get(protocolName)
	if !ok {
		return fmt.Errorf("protocol %q not found", protocolName)
	}

	index, err := GetProtocolIndex(ctx, protocol.Name, WithIndexer(m.Indexer), WithCacheFS(m.CacheFS))
	if err != nil {
		return fmt.Errorf("getting protocol index: %w", err)
	}

	if m.indexes == nil {
		m.indexes = make(map[string]Index)
	}

	if m.lookup == nil {
		m.lookup = make(map[string]*Page)
	}

	m.indexes[protocol.Name] = index

	type Payload struct {
		Name string
		UUID string
	}

	ch, mu := make(chan Payload), new(sync.Mutex)

	worker := func(ctx context.Context, wg *sync.WaitGroup, i int) {

		log := zerolog.Ctx(ctx).With().Int("worker", i).Logger()

		log.Debug().Msg("starting worker")

		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("stopping worker: context canceled")
				return
			case payload, ok := <-ch:
				if !ok {
					log.Debug().Msg("stopping worker")
					return
				}
				page, err := GetProtocolDocumentationPage(ctx,
					protocol.Name,
					payload.Name,
					payload.UUID,
					WithIndexer(m.Indexer),
					WithCacheFS(m.CacheFS),
				)
				if err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("getting protocol documentation page for %q: %w", payload.Name, err))
					mu.Unlock()
					continue
				}
				mu.Lock()
				pages = append(pages, page)
				mu.Unlock()
			}
		}
	}

	wg := new(sync.WaitGroup)

	for i := 0; i < 16; i++ {
		wg.Add(1)
		i := i
		go worker(ctx, wg, i)
	}

	index.EachSelf(func(name, value string) bool {
		if name, ok := ExtractName(name); ok {
			ch <- Payload{Name: name, UUID: value}
		}
		return true
	})

	close(ch)
	wg.Wait()

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].UUID < pages[j].UUID
	})

	for _, page := range pages {
		if _, ok := m.lookup[protocol.Name+"/"+page.UUID]; ok {
			continue
		}
		m.pages, m.lookup[protocol.Name+"/"+page.UUID] = append(m.pages, page), page
	}

	return errors.Join(errs...)
}
