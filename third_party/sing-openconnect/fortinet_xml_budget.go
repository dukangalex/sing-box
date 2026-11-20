package openconnect

import (
	"bytes"
	"fmt"
	"strings"
)

// EXPERIMENTAL APPLICATION RESOURCE POLICY, not a Fortinet protocol maximum.
// This preflight runs before encoding/xml can allocate an entire start token.
// Keep body bytes <= existing HTTP limit, total markup constructs <= 1000000, lexical chunks <= the existing HTTP body limit,
// attributes <= 65536 and semantic work units <= 65536. No truncation or
// deduplication: inputs outside the budget fail explicitly, with no partial cfg.
// These values require device measurement and real-server compatibility review
// before any promotion. The known 4097-domain and 64KiB-ignored-value fixtures
// stay supported; the existing 16 MiB body limit remains an application limit, not a protocol rule.
const fortinetBudgetAttributes = 65536
const fortinetBudgetMarkup = 1000000
const fortinetBudgetLexicalBytes = fortinetMaximumAuthenticationBody
const fortinetBudgetSemantic = 65536

func preflightFortinetXMLBudget(b []byte) error {
	if len(b) > fortinetMaximumAuthenticationBody {
		return fmt.Errorf("Fortinet XML resource budget: body bytes")
	}
	attrs, markup, depth := 0, 0, 0
	for i := 0; i < len(b); {
		j := bytes.IndexByte(b[i:], '<')
		if j < 0 {
			if len(b)-i > fortinetBudgetLexicalBytes {
				return fmt.Errorf("Fortinet XML resource budget: lexical bytes")
			}
			return nil
		}
		if j > fortinetBudgetLexicalBytes {
			return fmt.Errorf("Fortinet XML resource budget: lexical bytes")
		}
		tokenStart := i + j
		i += j
		markup++
		if markup > fortinetBudgetMarkup {
			return fmt.Errorf("Fortinet XML resource budget: markup constructs")
		}
		// XML validation remains with encoding/xml. These skips allocate nothing.
		// Unterminated markup is an error now, rather than allocating its contents.
		var terminator []byte
		switch {
		case bytes.HasPrefix(b[i:], []byte("<!--")):
			terminator = []byte("-->")
		case bytes.HasPrefix(b[i:], []byte("<![CDATA[")):
			terminator = []byte("]]>")
		case bytes.HasPrefix(b[i:], []byte("<?")):
			terminator = []byte("?>")
		}
		if terminator != nil {
			end := bytes.Index(b[i+2:], terminator)
			if end < 0 {
				return fmt.Errorf("unterminated Fortinet XML markup")
			}
			if 2+end+len(terminator) > fortinetBudgetLexicalBytes {
				return fmt.Errorf("Fortinet XML resource budget: lexical bytes")
			}
			i += 2 + end + len(terminator)
			continue
		}
		i++
		if i == len(b) {
			return fmt.Errorf("unterminated Fortinet XML tag")
		}
		directive := b[i] == '!'
		endTag := b[i] == '/'
		quote := byte(0)
		brackets := 0
		closed := false
		for ; i < len(b); i++ {
			if i-tokenStart >= fortinetBudgetLexicalBytes {
				return fmt.Errorf("Fortinet XML resource budget: lexical bytes")
			}
			if directive && quote == 0 && bytes.HasPrefix(b[i:], []byte("<!--")) {
				end := bytes.Index(b[i+4:], []byte("-->"))
				if end < 0 {
					return fmt.Errorf("unterminated Fortinet XML directive comment")
				}
				i += 4 + end + 2
				continue
			}
			c := b[i]
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '\'' || c == '"' {
				quote = c
				continue
			}
			if directive {
				if c == '[' {
					brackets++
				}
				if c == ']' && brackets > 0 {
					brackets--
				}
			}
			if c == '>' && brackets == 0 {
				if !directive {
					if endTag {
						if depth > 0 {
							depth--
						}
					} else {
						k := i - 1
						for k >= 0 && (b[k] == ' ' || b[k] == '\t' || b[k] == '\r' || b[k] == '\n') {
							k--
						}
						if k < 0 || b[k] != '/' {
							depth++
							if depth > 256 {
								return fmt.Errorf("Fortinet XML exceeds nesting limit")
							}
						}
					}
				}
				i++
				closed = true
				break
			}
			if !directive && !endTag && c == '=' {
				attrs++
				if attrs > fortinetBudgetAttributes {
					return fmt.Errorf("Fortinet XML resource budget: attributes")
				}
			}
		}
		if !closed {
			return fmt.Errorf("unterminated Fortinet XML tag")
		}
	}
	return nil
}

// Charge before downstream split-DNS map/slice allocations. Counts entries even
// if normalization would discard duplicates; duplicate semantics stay unchanged
// for accepted configurations. Does not scan irrelevant attributes or strings.
func fortinetSemanticCost(node fortinetXMLNode) int {
	cost := 1
	if node.name.Local == "split-dns" {
		if s, ok := node.attribute("domains"); ok && strings.TrimSpace(s) != "" {
			cost += 1 + strings.Count(s, ",")
		}
	}
	return cost
}
