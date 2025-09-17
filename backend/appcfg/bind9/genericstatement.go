package bind9config

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
)

var _ formattedElement = (*GenericClauseContents)(nil)

// The peeking iterator is used to serialize the GenericClauseContents.
// It implements the pattern whereby it is possible to get the next token
// from the sequence without consuming it. Serialization can be then delegated
// to a separate function recursively. This function can later peek and consume
// the token.
type peekingIterator struct {
	tokens []string
	index  int
}

// Instantiates a new peeking iterator with the specified tokens.
func newPeekingIterator(tokens []string) *peekingIterator {
	return &peekingIterator{
		tokens: tokens,
		index:  0,
	}
}

// Consumes the next token from the sequence.
func (i *peekingIterator) pop() {
	if i.index >= len(i.tokens) {
		return
	}
	i.index++
}

// Peeks the next token from the sequence without consuming it.
// It returns false as the second argument if it was the last
// token in the sequence.
func (i *peekingIterator) peek() (string, bool) {
	if i.index >= len(i.tokens) {
		return "", false
	}
	token := i.tokens[i.index]
	return token, true
}

// GenericClauseContents is used to parse any type of contents. It is
// used for parsing the configuration elements that are not explicitly
// supported in this parser. It consumes the tokens and stores them in the
// tokens field. They can be later serialized into a string representation
// with correct indentation and formatting.
type GenericClauseContents struct {
	tokens []string
}

// Parses the contents of a generic clause.
func (b *GenericClauseContents) Parse(lex *lexer.PeekingLexer) error {
	cnt := 0
	for {
		// Get the next token without consuming it.
		token := lex.Peek()
		switch {
		case token.EOF():
			// The end of the statement contents.
			return nil
		case token.Value == "{":
			// Opening new sub-statement. Increase the
			// counter to keep track of the nesting level.
			cnt++
		case token.Value == "}":
			// Closing sub-statement. Decrease the counter
			// to keep track of the nesting level.
			cnt--
			if cnt < 0 {
				// Extraneous closing brace found.
				return nil
			}
		}
		// Consume the token.
		if token.Type == bind9Lexer.Symbols()["String"] {
			b.tokens = append(b.tokens, fmt.Sprintf(`"%s"`, lex.Next().Value))
		} else {
			b.tokens = append(b.tokens, lex.Next().Value)
		}
	}
}

// Returns the serialized BIND 9 configuration for the generic clause contents.
func (b *GenericClauseContents) getFormattedOutput(filter *Filter) formatterOutput {
	scope := b.createScopeFromTokens(newPeekingIterator(b.tokens))
	return scope
}

// Iterates over a sequence of parsed tokens and attempts to create a
// formatted clause from it. This function can be called recursively.
func (b *GenericClauseContents) createClauseFromTokens(iter *peekingIterator) formatterOutput {
	clause := newFormatterClause()
	for {
		token, ok := iter.peek()
		if !ok {
			return nil
		}
		switch token {
		case "{":
			iter.pop()
			clause.add(b.createScopeFromTokens(iter))
		case "}":
			return clause
		case ";":
			iter.pop()
			return clause
		default:
			iter.pop()
			clause.addToken(token)
		}
	}
}

// Iterates over a sequence of parsed tokens and attempts to create a
// formatted scope from it. This function can be called recursively.
func (b *GenericClauseContents) createScopeFromTokens(iter *peekingIterator) formatterOutput {
	scope := newFormatterScope()
	for {
		token, ok := iter.peek()
		if !ok {
			break
		}
		switch token {
		case "{":
			iter.pop()
			scope.add(b.createScopeFromTokens(iter))
		case "}":
			iter.pop()
			return scope
		case ";":
			iter.pop()
		default:
			c := b.createClauseFromTokens(iter)
			if c == nil {
				return scope
			}
			scope.add(c)
		}
	}
	return scope
}
