/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

// Package nquads parses N-Quads (https://www.w3.org/TR/n-quads/)
package nquads

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/iand/gordf"
)

// A ParseError is returned for parsing errors.
// The first line is 1.  The first column is 0.
type ParseError struct {
	Line   int   // Line where the error occurred
	Column int   // Column (rune index) where the error occurred
	Err    error // The actual error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// These are the errors that can be returned in ParseError.Error
var (

	// ErrUnexpectedCharacter is the error returned when a general syntax error is encountered.
	ErrUnexpectedCharacter = errors.New("unexpected character")

	// ErrInvalidCodepointExpression is the error returned when an IRI or literal is encountered with an invalid
	// unicode encoding of the form \Uxxxxxxxx or \uxxxx is encountered.
	ErrInvalidCodepointExpression = errors.New("invalid unicode codepoint expression")

	// ErrUnexpectedEOF is the error returned when an EOF was encountered while reading a quad.
	ErrUnexpectedEOF = io.ErrUnexpectedEOF

	// ErrUnterminatedQuad is the error returned when a quad appears to be missing the final period.
	ErrUnterminatedQuad = errors.New("unterminated quad, expecting '.'")

	// ErrRelativeIRI is the error returned when a relative IRI is encountered. All IRIs in an N-Quads document must
	// be written as absolute IRIs.
	ErrRelativeIRI = errors.New("relative IRI")
)

type Reader struct {
	line   int
	column int
	r      *bufio.Reader
	buf    bytes.Buffer
	err    error
	q      Quad
}

// A Quad consists of a subject, predicate, object and graph
type Quad struct {
	S rdf.Term
	P rdf.Term
	O rdf.Term
	G rdf.Term
}

func (q Quad) String() string {
	if q.G.Kind == rdf.UnknownTerm {
		return fmt.Sprintf("%s %s %s .", q.S.String(), q.P.String(), q.O.String())
	}
	return fmt.Sprintf("%s %s %s %s .", q.S.String(), q.P.String(), q.O.String(), q.G.String())
}

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

// wrap creates a new ParseError using err, annotating it with the current column and line number.
func (r *Reader) wrap(err error) error {
	return &ParseError{
		Line:   r.line,
		Column: r.column,
		Err:    err,
	}
}

// Err returns any error encountered while reading. If Err is non-nil then Next will always return false.
func (r *Reader) Err() error {
	return r.err
}

// Quad returns the last quad read
func (r *Reader) Quad() Quad {
	return r.q
}

// Next attempts to read the next quad from the underlying reader. It returns false if no quad could be read which
// may indicate an error has occurred or the end of the input stream has been reached.
func (r *Reader) Next() bool {
	if r.err != nil {
		return false
	}

	r.q = Quad{}
	r.line++
	r.column = -1

	var err error
	r1 := '\n'
	for r1 == '\n' {
		r1, err = r.skipWhitespace()
		if err != nil {
			if err == io.EOF {
				return false
			}
			r.err = err
			return false
		}

		if r1 == '#' {
			r1, err = r.skipRestOfLine()
			if err != nil {
				if err == io.EOF {
					return false
				}
				r.err = err
				return false
			}
		}
	}

	if err := r.r.UnreadRune(); err != nil {
		r.err = err
		return false
	}

	// Subject
	term, err := r.parseIriOrBlankNode()
	if err != nil {
		r.err = err
		return false
	}
	if term.Kind == rdf.IRITerm && !isAbsoluteIRI(term.Value) {
		r.err = r.wrap(ErrRelativeIRI)
		return false
	}
	r.q.S = term

	// Property
	term, err = r.parseIriOrBlankNode()
	if err != nil {
		r.err = err
		return false
	}
	if term.Kind == rdf.IRITerm && !isAbsoluteIRI(term.Value) {
		r.err = r.wrap(ErrRelativeIRI)
		return false
	}
	r.q.P = term

	// Object
	term, err = r.parseAnyTerm()
	if err != nil {
		r.err = err
		return false
	}
	if term.Kind == rdf.IRITerm && !isAbsoluteIRI(term.Value) {
		r.err = r.wrap(ErrRelativeIRI)
		return false
	} else if term.Kind == rdf.LiteralTerm && term.Datatype != "" && !isAbsoluteIRI(term.Datatype) {
		r.err = r.wrap(ErrRelativeIRI)
		return false
	}
	r.q.O = term

	// Graph or end
	end, term, err := r.parseIriOrBlankNodeOrEndTriple()
	if err != nil {
		r.err = err
		return false
	}

	if end {
		r.err = r.expectCommentOrEndOfLine()
		return r.err == nil
	}

	r.q.G = term
	err = r.readEndQuad()
	if err != nil {
		r.err = err
		return false
	}
	r.err = r.expectCommentOrEndOfLine()
	return r.err == nil
}

// readRune reads one rune from r, folding \r\n to \n and keeping track
// of how far into the line we have read.  r.column will point to the start
// of this rune, not the end of this rune.
func (r *Reader) readRune() (rune, error) {
	r1, _, err := r.r.ReadRune()

	// Handle \r\n here.  We make the simplifying assumption that
	// anytime \r is followed by \n that it can be folded to \n.
	// We will not detect files which contain both \r\n and bare \n.
	if r1 == '\r' {
		r1, _, err = r.r.ReadRune()
		if err == nil {
			if r1 != '\n' {
				if err := r.r.UnreadRune(); err != nil {
					return r1, err
				}
				r1 = '\r'
			}
		}
	}
	r.column++
	return r1, err
}

// unreadRune puts the last rune read from r back.
func (r *Reader) unreadRune() error {
	if err := r.r.UnreadRune(); err != nil {
		return err
	}
	r.column--
	return nil
}

func (r *Reader) parseIRI() (term rdf.Term, err error) {
	for {
		r1, err := r.readRune()
		if err != nil {
			if err == io.EOF {
				return term, r.wrap(ErrUnexpectedEOF)
			}
			return term, err
		}

		if r1 <= 0x20 || r1 == '<' || r1 == '"' || r1 == '{' || r1 == '}' || r1 == '|' || r1 == '^' || r1 == '`' {
			return term, r.wrap(ErrUnexpectedCharacter)
		} else if r1 == '>' {
			if r.buf.Len() == 0 {
				return term, r.wrap(ErrUnexpectedCharacter)
			}
			return rdf.IRI(r.buf.String()), nil

		} else if r1 == '\\' {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return term, r.wrap(ErrUnexpectedEOF)
				}
				return term, err
			}
			switch r1 {
			case 'u', 'U':
				size := 4
				if r1 == 'U' {
					size = 8
				}
				codepoint := rune(0)

				for i := size - 1; i >= 0; i-- {
					r1, err = r.readRune()

					if err != nil {
						if err == io.EOF {
							return term, r.wrap(ErrUnexpectedEOF)
						}
						return term, err
					}

					if r1 >= '0' && r1 <= '9' {
						codepoint += (1 << uint32(4*i)) * (r1 - '0')
					} else if r1 >= 'a' && r1 <= 'f' {
						codepoint += (1 << uint32(4*i)) * (r1 - 'a' + 10)
					} else if r1 >= 'A' && r1 <= 'F' {
						codepoint += (1 << uint32(4*i)) * (r1 - 'A' + 10)
					} else {
						return term, r.wrap(ErrInvalidCodepointExpression)
					}

				}
				r.buf.WriteRune(codepoint)
			default:
				return term, r.wrap(ErrUnexpectedCharacter)
			}

		} else {
			r.buf.WriteRune(r1)
		}

	}
}

func (r *Reader) parseBlankNode() (rdf.Term, error) {
	r1, err := r.readRune()
	if err != nil {
		if err == io.EOF {
			return rdf.Term{}, r.wrap(ErrUnexpectedEOF)
		}
		return rdf.Term{}, err
	}

	if r1 != ':' {
		return rdf.Term{}, r.wrap(ErrUnexpectedCharacter)
	}

	r1, err = r.readRune()
	if err != nil {
		if err == io.EOF {
			return rdf.Term{}, r.wrap(ErrUnexpectedEOF)
		}
		return rdf.Term{}, err
	}
	if !(isPnCharsU(r1) || isNumeral(r1)) {
		return rdf.Term{}, r.wrap(ErrUnexpectedCharacter)
	}
	r.buf.WriteRune(r1)

	for {
		r1, err = r.readRune()
		if err != nil {
			if err == io.EOF {
				return rdf.Term{}, r.wrap(ErrUnexpectedEOF)
			}
			return rdf.Term{}, err
		}

		if isPnChars(r1) {
			r.buf.WriteRune(r1)
		} else if isSpace(r1) {
			return rdf.Blank(r.buf.String()), nil
		} else if r1 == '.' {
			err := r.unreadRune()
			if err != nil {
				return rdf.Term{}, nil
			}
			// period is not allowed at the end of a blank node
			next, err := r.r.Peek(2)
			if err == io.EOF {
				// period is the last character in the file so must be a triple terminator
				return rdf.Blank(r.buf.String()), nil
			}

			if next[1] == ' ' || next[1] == '\t' || next[1] == '\n' || next[1] == '\r' {
				// period is not part of the blank node
				return rdf.Blank(r.buf.String()), nil
			}

			if _, err := r.readRune(); err != nil {
				return rdf.Term{}, err
			}
			r.buf.WriteRune(r1)

		} else {
			return rdf.Term{}, r.wrap(ErrUnexpectedCharacter)
		}

	}
}

func (r *Reader) parseLiteral() (term rdf.Term, err error) {
	for {
		r1, err := r.readRune()
		if err != nil {
			if err == io.EOF {
				return term, r.wrap(ErrUnexpectedEOF)
			}
			return term, err
		}
		switch r1 {
		case '"':
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return term, r.wrap(ErrUnexpectedEOF)
				}
				return term, err
			}

			switch r1 {

			case '.', ' ', '\t':
				if err := r.unreadRune(); err != nil {
					return term, r.wrap(err)
				}
				return rdf.Literal(r.buf.String()), nil
			case '@':
				value := r.buf.String()
				r.buf.Reset()

				major := true
				for {
					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return term, r.wrap(ErrUnexpectedEOF)
						}
						return term, err
					}
					if r1 == '.' || isSpace(r1) {
						if r.buf.Len() == 0 {
							return term, r.wrap(ErrUnexpectedCharacter)
						}
						if err := r.unreadRune(); err != nil {
							return term, r.wrap(err)
						}
						return rdf.LiteralWithLanguage(value, r.buf.String()), nil
					}
					if major {
						if isAlpha(r1) {
							r.buf.WriteRune(r1)
						} else if r1 == '-' {
							r.buf.WriteRune(r1)
							major = false // switch to language subtype
						} else {
							return term, r.wrap(ErrUnexpectedCharacter)
						}
					} else {
						if isAlpha(r1) || isNumeral(r1) {
							r.buf.WriteRune(r1)
						} else {
							return term, r.wrap(ErrUnexpectedCharacter)
						}
					}
				}
			case '^':
				value := r.buf.String()
				r.buf.Reset()

				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return term, r.wrap(ErrUnexpectedEOF)
					}
					return term, err
				}
				if r1 != '^' {
					return term, r.wrap(ErrUnexpectedCharacter)
				}

				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return term, r.wrap(ErrUnexpectedEOF)
					}
					return term, err
				}
				if r1 != '<' {
					return term, r.wrap(ErrUnexpectedCharacter)
				}

				// Read an IRI
				for {
					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return term, r.wrap(ErrUnexpectedEOF)
						}
						return term, err
					}
					if r1 == '>' {
						if r.buf.Len() == 0 {
							return term, r.wrap(ErrUnexpectedCharacter)
						}
						return rdf.LiteralWithDatatype(value, r.buf.String()), nil
					} else if r1 < 0x20 || r1 > 0x7E || r1 == ' ' || r1 == '<' || r1 == '"' {
						return term, r.wrap(ErrUnexpectedCharacter)
					}
					r.buf.WriteRune(r1)
				}

			}
			return term, r.wrap(ErrUnexpectedCharacter)

		case '\\':
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return term, r.wrap(ErrUnexpectedEOF)
				}
				return term, err
			}
			switch r1 {
			case '\\', '"':
			case 't':
				r1 = '\t'
			case 'r':
				r1 = '\r'
			case 'n':
				r1 = '\n'
			case 'b':
				r1 = '\b'
			case 'f':
				r1 = '\f'
			case 'u', 'U':
				size := 4
				if r1 == 'U' {
					size = 8
				}

				codepoint := rune(0)

				for i := size - 1; i >= 0; i-- {
					r1, err = r.readRune()

					if err != nil {
						if err == io.EOF {
							return term, r.wrap(ErrUnexpectedEOF)
						}
						return term, err
					}

					if r1 >= '0' && r1 <= '9' {
						codepoint += (1 << uint32(4*i)) * (r1 - '0')
					} else if r1 >= 'a' && r1 <= 'f' {
						codepoint += (1 << uint32(4*i)) * (r1 - 'a' + 10)
					} else if r1 >= 'A' && r1 <= 'F' {
						codepoint += (1 << uint32(4*i)) * (r1 - 'A' + 10)
					} else {
						return term, r.wrap(ErrInvalidCodepointExpression)
					}

				}
				r1 = codepoint

			default:
				return term, r.wrap(ErrUnexpectedCharacter)
			}
		}
		r.buf.WriteRune(r1)
	}
}

func (r *Reader) parseIriOrBlankNode() (term rdf.Term, err error) {
	r.buf.Reset()

	r1, err := r.skipWhitespace()
	if err != nil {
		return term, err
	}
	switch r1 {
	case '<':
		// Read an IRI
		return r.parseIRI()
	case '_':
		// Read a blank node
		return r.parseBlankNode()
	default:
		// TODO: raise error, unexpected character
		return term, r.wrap(ErrUnexpectedCharacter)

	}
}

func (r *Reader) parseAnyTerm() (term rdf.Term, err error) {
	r.buf.Reset()

	r1, err := r.skipWhitespace()
	if err != nil {
		return term, err
	}
	switch r1 {
	case '<':
		// Read an IRI
		return r.parseIRI()
	case '_':
		// Read a blank node
		return r.parseBlankNode()
	case '"':
		// Read a literal
		return r.parseLiteral()
	default:
		// TODO: raise error, unexpected character
		return term, r.wrap(ErrUnexpectedCharacter)

	}
}

func (r *Reader) parseIriOrBlankNodeOrEndTriple() (bool, rdf.Term, error) {
	r.buf.Reset()

	r1, err := r.skipWhitespace()
	if err != nil {
		return false, rdf.Term{}, err
	}
	switch r1 {
	case '<':
		// Read an IRI
		term, err := r.parseIRI()
		return false, term, err
	case '_':
		// Read a blank node
		term, err := r.parseBlankNode()
		return false, term, err
	case '.':
		// End of triple
		return true, rdf.Term{}, nil
	default:
		return false, rdf.Term{}, r.wrap(ErrUnexpectedCharacter)
	}
}

func (r *Reader) readEndQuad() (err error) {
	r1, err := r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return r.wrap(ErrUnterminatedQuad)
		}
		return err
	}

	if r1 != '.' {
		return r.wrap(ErrUnexpectedCharacter)
	}

	return nil
}

func (r *Reader) skipWhitespace() (r1 rune, err error) {
	r1, err = r.readRune()
	if err != nil {
		return r1, err
	}

	for isSpace(r1) {
		r1, err = r.readRune()
		if err != nil {
			return r1, err
		}
	}

	return r1, nil
}

func (r *Reader) skipRestOfLine() (r1 rune, err error) {
	r1, err = r.readRune()
	if err != nil {
		return r1, err
	}

	for r1 != '\n' {
		r1, err = r.readRune()
		if err != nil {
			return r1, err
		}
	}
	r.line++
	r.column = 0

	// r1 is now the newline
	return r1, nil
}

func (r *Reader) expectCommentOrEndOfLine() error {
	r1, err := r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	if r1 == '#' {
		_, err = r.skipRestOfLine()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		return nil
	}

	if r1 != '\n' {
		return r.wrap(ErrUnexpectedCharacter)
	}

	return nil
}

func isPnCharsBase(r rune) bool {
	if isAlpha(r) {
		return true
	}

	if r >= 0x00C0 && r <= 0x00D6 {
		return true
	}
	if r >= 0x00D8 && r <= 0x00F6 {
		return true
	}
	if r >= 0x00F8 && r <= 0x02FF {
		return true
	}
	if r >= 0x0370 && r <= 0x037D {
		return true
	}
	if r >= 0x037F && r <= 0x1FFF {
		return true
	}
	if r >= 0x200C && r <= 0x200D {
		return true
	}
	if r >= 0x2070 && r <= 0x218F {
		return true
	}
	if r >= 0x2C00 && r <= 0x2FEF {
		return true
	}
	if r >= 0x3001 && r <= 0xD7FF {
		return true
	}
	if r >= 0xF900 && r <= 0xFDCF {
		return true
	}
	if r >= 0xFDF0 && r <= 0xFFFD {
		return true
	}
	if r >= 0x10000 && r <= 0xEFFFF {
		return true
	}

	return false
}

func isPnCharsU(r rune) bool {
	if r == '_' || r == ':' {
		return true
	}
	if isPnCharsBase(r) {
		return true
	}
	return false
}

func isPnChars(r rune) bool {
	if r == '-' || r == 0x00B7 {
		return true
	}
	if isNumeral(r) {
		return true
	}
	if isPnCharsU(r) {
		return true
	}
	if r >= 0x0300 && r <= 0x036F {
		return true
	}
	if r >= 0x203F && r <= 0x2040 {
		return true
	}

	return false
}

func isNumeral(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlpha(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isAbsoluteIRI(s string) bool {
	// lightweight test
	return strings.Contains(s, ":")
}
