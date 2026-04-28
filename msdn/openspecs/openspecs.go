package openspecs

import (
	"context"
)

const (
	// OpenSpecsBaseURL is the base URL for Microsoft Open Specifications.
	OpenSpecsBaseURL = "https://learn.microsoft.com/en-us/openspecs/"
	// WindowsProtocols is the identifier for the Windows Protocols section in Microsoft Open Specifications.
	WindowsProtocols = "windows_protocols"
	// ExchangeServerProtocols is the identifier for the Exchange Server Protocols
	// section in Microsoft Open Specifications.
	ExchangeServerProtocols = "exchange_server_protocols"
)

// GetProtocolIndexByID is a helper function that constructs the URL for a specific protocol index.
func GetProtocolIndexByID(ctx context.Context, protocolName, indexUUID string, opts ...ClientOption) (Index, error) {
	return MakeClient(opts...).GetProtocolIndexByID(ctx, protocolName, indexUUID)
}

func GetProtocolIndex(ctx context.Context, protocolName string, opts ...ClientOption) (Index, error) {
	return MakeClient(opts...).GetProtocolIndex(ctx, protocolName)
}

// GetProtocolDocumentationPage is a helper function that constructs the URL for a specific documentation page
// of a protocol in the Microsoft Open Specifications and retrieves the content of that page.
func GetProtocolDocumentationPage(ctx context.Context, protocolName, pageName, pageUUID string, opts ...ClientOption) (*Page, error) {
	return MakeClient(opts...).GetProtocolDocumentationPage(ctx, protocolName, pageName, pageUUID)
}

// MakeClient is a helper function that creates a new Client instance with the provided options.
// It takes a variable number of ClientOption functions, applies them to a new Client instance, and
// returns the configured Client.
func MakeClient(opts ...ClientOption) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ClientOption is a function type that defines an option for configuring the Client. It takes a pointer to a Client
// and modifies it according to the specific option being applied. This allows for flexible and extensible configuration
// of the Client when creating a new instance, enabling users to customize its behavior without needing to modify the Client's constructor directly.
type ClientOption func(*Client)

// WithCacheFS allows you to set a file-system cache for the Client. The root parameter specifies the directory
// where cached files will be stored. This can be used to persist cached data across runs of the program.
func WithCacheFS(root string) ClientOption {
	return func(c *Client) {
		c.Cache = NewCacheFS(root)
	}
}

// WithIndexer allows you to set a custom ProtocolIndexer for the Client.
// This can be used to provide a pre-populated index or to customize the indexing behavior.
func WithIndexer(indexer *ProtocolIndexer) ClientOption {
	return func(c *Client) {
		c.Indexer = indexer
	}
}
