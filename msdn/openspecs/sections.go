package openspecs

// sections.go contains the implementation of the PrefixSection type, which represents a section of documentation
// that can be identified by specific prefixes or contained strings in the documentation text. It also includes
// a predefined list of default PrefixSection instances that can be used to identify common sections in the Microsoft
// Open Specifications or similar documentation.

import "strings"

// PrefixSection represents a section of documentation that can be identified
// by specific prefixes or contained strings in the documentation text.
type PrefixSection struct {
	// Name is the name of the section, which serves as a heading for the content within that section.
	Name string
	// Prefixes is a list of string prefixes that can be used to identify the section in the documentation.
	Prefixes []string
	// Contains is a list of strings that can be used to identify the section if they are contained
	// anywhere in the documentation text.
	Contains []string
}

// Match checks if the given string matches any of the prefixes or contains any of the specified strings
// in the PrefixSection. It returns true if a match is found, and false otherwise.
func (p PrefixSection) Match(s string) bool {
	for _, prefix := range p.Prefixes {
		if strings.HasPrefix(strings.TrimSpace(s), prefix) {
			return true
		}
	}

	for _, contains := range p.Contains {
		if strings.Contains(strings.TrimSpace(s), contains) {
			return true
		}
	}

	return false
}

// DefaultPrefixes is a predefined list of PrefixSection instances that can be used to identify
// common sections in documentation based on their prefixes or contained strings.
// This list can be used as a default set of rules for categorizing sections in the Microsoft Open
// Specifications or similar documentation.
var DefaultPrefixes = []PrefixSection{
	{
		Name: "Call Processing",
		Prefixes: []string{
			"When processing",
			"The following statements define the sequence",
			"Message processing for",
			"The processing for this method",
			"The server MUST",
			"The RPC server MUST",
			"The behavior required when receiving",
			"The following are semantic checks",
			"Processing rules",
			"The processing rules",
			"Processing:",
			"Processing instructions:",
			"While processing this",
			"Sequential Processing Rules:",
			"When W32Time",
			"The CA server MUST",
			"The following processing rules apply",
			"In order to perform",
			"The following is an overview",
		},
	},
	{
		Name: "Call Response",
		Prefixes: []string{
			"In response",
			"The response of the server",
			"When a RAZA server receives this message",
			"On receipt of this message",
			"When this method is",
		},
	},
	{
		Name: "Call Received",
		Prefixes: []string{

			"Upon receiving",
			"On receiving",
			"When the server receives",
			"After receiving this message",
			"Upon receipt",
		},
	},
	{
		Name: "Exceptions Thrown",
		Contains: []string{
			"Exceptions Thrown",
		},
	},
	{
		Name: "Error Codes",
		Prefixes: []string{
			"Error Codes",
		},
	},
	{
		Name: "Server Operations",
		Contains: []string{
			"Server Operations",
		},
	},
	{
		Name: "Call Definitions",
		Prefixes: []string{
			"The following definitions",
			"These definitions",
		},
	},
	{
		Name: "Call Validations",
		Prefixes: []string{
			"The following validation",
		},
	},
	{
		Name: "Security",
		Prefixes: []string{
			"The security principal",
			"This method obtains the identity",
		},
	},
	{
		Name: "Normative",
		Prefixes: []string{
			"Sections",
		},
	},
}
