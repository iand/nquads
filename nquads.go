/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

// Package nquads parses N-Quads
package nquads

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode"
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

// These are the errors that can be returned in ParseError.Error
var (
	ErrUnexpectedCharacter = errors.New("unexpected character")
	ErrUnexpectedEOF       = errors.New("unexpected end of file")
	ErrTermCount           = errors.New("wrong number of terms in line")
	ErrUnterminatedIri     = errors.New("unterminated IRI, expecting '>'")
	ErrUnterminatedLiteral = errors.New("unterminated literal, expecting '\"'")
	ErrUnterminatedQuad    = errors.New("unterminated quad, expecting '.'")
)

type Reader struct {
	line   int
	column int
	r      *bufio.Reader
	buf    bytes.Buffer
	err    error
	q      Quad
}

// A Quad consists of a graph, subject, predicate and object
type Quad struct {
	G RdfTerm
	S RdfTerm
	P RdfTerm
	O RdfTerm
}

func (q Quad) String() string {
	return fmt.Sprintf("%s %s %s %s .", q.G, q.S, q.P, q.O)
}

// An RdfTerm represents one of Iri, Blank Node or Literal
type RdfTerm struct {
	Value    string
	Language string
	DataType string
	TermType int
}

func (t RdfTerm) String() string {
	switch t.TermType {
	case RdfIri:
		return fmt.Sprintf("<%s>", t.Value)

	case RdfBlank:
		return fmt.Sprintf("_:%s", t.Value)
	case RdfLiteral:
		if t.Language != "" {
			return fmt.Sprintf("\"%s\"@%s", t.Value, t.Language)
		} else if t.DataType != "" {
			return fmt.Sprintf("\"%s\"^^<%s>", t.Value, t.DataType)
		} else {
			return fmt.Sprintf("\"%s\"", t.Value)
		}
	}

	return "[unknown type]"
}

func (t RdfTerm) IsIRI() bool {
	return t.TermType == RdfIri
}

func (t RdfTerm) IsBlank() bool {
	return t.TermType == RdfBlank
}

func (t RdfTerm) IsLiteral() bool {
	return t.TermType == RdfLiteral
}

func (t RdfTerm) IsTypedLiteral() bool {
	return t.TermType == RdfLiteral && t.DataType != ""
}

func (t RdfTerm) IsLanguageLiteral() bool {
	return t.TermType == RdfLiteral && t.Language != ""
}

// Constants for types of RdfTerm
const (
	RdfUnknown = iota
	RdfIri
	RdfBlank
	RdfLiteral
)

// NewReader returns a new Reader that reads from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: bufio.NewReader(r),
	}
}

// error creates a new ParseError based on err.
func (r *Reader) error(err error) error {
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

	r1, err := r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return false
		}
		r.err = err
		return false
	}

	for r1 == '#' {
		for {
			r1, err = r.readRune()
			if err != nil {
				r.err = err
				return false
			}
			if r1 == '\n' {
				break
			}
		}
		r1, err = r.skipWhitespace()
		if err != nil {
			r.err = err
			return false
		}

	}

	if err := r.r.UnreadRune(); err != nil {
		r.err = err
		return false
	}

	termCount := 0
	for {
		haveTerm, term, err := r.parseTerm()
		if haveTerm {
			termCount++
			switch termCount {
			case 1:
				r.q.G = term
				err = r.expectWhitespace()
				if err != nil {
					r.err = err
					return false
				}
			case 2:
				r.q.S = term
				err = r.expectWhitespace()
				if err != nil {
					r.err = err
					return false
				}
			case 3:
				r.q.P = term
				err = r.expectWhitespace()
				if err != nil {
					r.err = err
					return false
				}
			case 4:
				r.q.O = term

				err = r.readEndQuad()
				if err != nil {
					r.err = err
					return false
				}

				return true
			default:
				r.err = &ParseError{
					Line:   r.line,
					Column: r.column,
					Err:    ErrTermCount,
				}
				return false
			}

		}
		if err != nil {
			r.err = err
			return false
		}
	}

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

func (r *Reader) parseTerm() (haveField bool, term RdfTerm, err error) {
	r.buf.Reset()

	r1, err := r.skipWhitespace()
	if err != nil {
		return false, term, err
	}
	switch r1 {
	case '<':
		// Read an IRI
		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			if r1 == '>' {
				if r.buf.Len() == 0 {
					return false, term, r.error(ErrUnexpectedCharacter)
				}
				return true, RdfTerm{Value: r.buf.String(), TermType: RdfIri}, nil
			} else if r1 < 0x20 || r1 > 0x7E || r1 == ' ' || r1 == '<' || r1 == '"' {
				return false, term, r.error(ErrUnexpectedCharacter)
			}
			r.buf.WriteRune(r1)
		}
	case '_':
		// Read a blank node
		r1, err = r.readRune()
		if err != nil {
			if err == io.EOF {
				return false, term, r.error(ErrUnexpectedEOF)
			}
			return false, term, err
		}

		if r1 != ':' {
			return false, term, r.error(ErrUnexpectedCharacter)
		}

		r1, err = r.readRune()
		if err != nil {
			if err == io.EOF {
				return false, term, r.error(ErrUnexpectedEOF)
			}
			return false, term, err
		}
		if !((r1 >= 'a' && r1 <= 'z') || (r1 >= 'A' && r1 <= 'Z')) {
			return false, term, r.error(ErrUnexpectedCharacter)
		}
		r.buf.WriteRune(r1)

		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			if !((r1 >= 'a' && r1 <= 'z') || (r1 >= 'A' && r1 <= 'Z') || (r1 >= '0' && r1 <= '9')) {
				if r1 == '.' || unicode.IsSpace(r1) {
					if err := r.unreadRune(); err != nil {
						return false, term, r.error(err)
					}
					return true, RdfTerm{Value: r.buf.String(), TermType: RdfBlank}, nil
				}
				return false, term, r.error(ErrUnexpectedCharacter)
			}
			r.buf.WriteRune(r1)
		}

	case '"':
		// Read a literal
		for {
			r1, err = r.readRune()
			if err != nil {
				if err == io.EOF {
					return false, term, r.error(ErrUnexpectedEOF)
				}
				return false, term, err
			}
			switch r1 {
			case '"':
				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return false, term, r.error(ErrUnexpectedEOF)
					}
					return false, term, err
				}

				switch r1 {

				case '.', ' ', '\t':
					if err := r.unreadRune(); err != nil {
						return false, term, r.error(err)
					}
					return true, RdfTerm{Value: r.buf.String(), TermType: RdfLiteral}, nil
				case '@':
					tmpterm := RdfTerm{Value: r.buf.String(), TermType: RdfLiteral}
					r.buf.Reset()

					for {
						r1, err = r.readRune()
						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}
						if r1 == '.' || r1 == ' ' || r1 == '\t' {
							if r.buf.Len() == 0 {
								return false, term, r.error(ErrUnexpectedCharacter)
							}
							tmpterm.Language = r.buf.String()
							return true, tmpterm, nil
						}
						if r1 == '-' || (r1 >= 'a' && r1 <= 'z') || (r1 >= '0' && r1 <= '9') {
							r.buf.WriteRune(r1)
						} else {
							return false, term, r.error(ErrUnexpectedCharacter)
						}
					}
				case '^':
					tmpterm := RdfTerm{Value: r.buf.String(), TermType: RdfLiteral}
					r.buf.Reset()

					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return false, term, r.error(ErrUnexpectedEOF)
						}
						return false, term, err
					}
					if r1 != '^' {
						return false, term, r.error(ErrUnexpectedCharacter)
					}

					r1, err = r.readRune()
					if err != nil {
						if err == io.EOF {
							return false, term, r.error(ErrUnexpectedEOF)
						}
						return false, term, err
					}
					if r1 != '<' {
						return false, term, r.error(ErrUnexpectedCharacter)
					}

					// Read an IRI
					for {
						r1, err = r.readRune()
						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}
						if r1 == '>' {
							if r.buf.Len() == 0 {
								return false, term, r.error(ErrUnexpectedCharacter)
							}
							tmpterm.DataType = r.buf.String()
							return true, tmpterm, nil
						} else if r1 < 0x20 || r1 > 0x7E || r1 == ' ' || r1 == '<' || r1 == '"' {
							return false, term, r.error(ErrUnexpectedCharacter)
						}
						r.buf.WriteRune(r1)
					}

				}
				return false, term, r.error(ErrUnexpectedCharacter)

			case '\\':
				r1, err = r.readRune()
				if err != nil {
					if err == io.EOF {
						return false, term, r.error(ErrUnexpectedEOF)
					}
					return false, term, err
				}
				switch r1 {
				case '\\', '"':
				case 't':
					r1 = '\t'
				case 'r':
					r1 = '\r'
				case 'n':
					r1 = '\n'
				case 'u', 'U':

					codepoint := rune(0)

					for i := 3; i >= 0; i-- {
						r1, err = r.readRune()

						if err != nil {
							if err == io.EOF {
								return false, term, r.error(ErrUnexpectedEOF)
							}
							return false, term, err
						}

						if r1 >= '0' && r1 <= '9' {
							codepoint += (1 << uint32(4*i)) * (r1 - '0')
						} else if r1 >= 'a' && r1 <= 'f' {
							codepoint += (1 << uint32(4*i)) * (r1 - 'a' + 10)
						} else if r1 >= 'A' && r1 <= 'F' {
							codepoint += (1 << uint32(4*i)) * (r1 - 'A' + 10)
						} else {
							return false, term, r.error(ErrUnexpectedCharacter)
						}

					}
					r1 = codepoint

				default:
					return false, term, r.error(ErrUnexpectedCharacter)
				}
			}
			r.buf.WriteRune(r1)
		}
	default:
		// TODO: raise error, unexpected character
		return false, term, r.error(ErrUnexpectedCharacter)

	}

}

func (r *Reader) readEndQuad() (err error) {
	r1, err := r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return r.error(ErrUnterminatedQuad)
		}
		return err
	}

	if r1 != '.' {
		return r.error(ErrUnexpectedCharacter)
	}

	r1, err = r.skipWhitespace()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	if r1 != '\n' {
		return r.error(ErrUnexpectedCharacter)
	}

	return nil

}

func (r *Reader) skipWhitespace() (r1 rune, err error) {
	r1, err = r.readRune()
	if err != nil {
		return r1, err
	}

	for r1 == ' ' || r1 == '\t' {
		r1, err = r.readRune()
		if err != nil {
			return r1, err
		}
	}

	return r1, nil

}

func (r *Reader) expectWhitespace() (err error) {
	r1, err := r.readRune()
	if err != nil {
		if err == io.EOF {
			return r.error(ErrUnexpectedEOF)
		}
		return err
	}
	if r1 != ' ' && r1 != '\t' {
		return r.error(ErrUnexpectedCharacter)
	}

	return nil
}
