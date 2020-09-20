/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

package nquads

import (
	"bytes"
	"strings"
	"testing"
)

var testCases = map[string]Quad{
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource1", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "http://example.org/resource2", TermType: RdfIri},
	},

	"<http://example.org/graph1> _:anon <http://example.org/property> <http://example.org/resource2> .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "anon", TermType: RdfBlank},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "http://example.org/resource2", TermType: RdfIri},
	},

	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> _:anon .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource1", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "anon", TermType: RdfBlank},
	},

	" <http://example.org/graph1> 	 <http://example.org/resource3> 	 <http://example.org/property>	 <http://example.org/resource2> 	.": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource3", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "http://example.org/resource2", TermType: RdfIri},
	},

	"<http://example.org/graph1> <http://example.org/resource7> <http://example.org/property> \"simple literal\" .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource7", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "simple literal", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource8> <http://example.org/property> "backslash:\\" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource8", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "backslash:\\", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource9> <http://example.org/property> "dquote:\"" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource9", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "dquote:\"", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource10> <http://example.org/property> "newline:\n" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource10", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "newline:\n", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource11> <http://example.org/property> "return\r" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource11", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "return\r", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource12> <http://example.org/property> "tab:\t" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource12", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "tab:\t", TermType: RdfLiteral}},

	`<http://example.org/graph1> <http://example.org/resource16> <http://example.org/property> "\u00E9" .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource16", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "\u00E9", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource30> <http://example.org/property> "chat"@fr .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource30", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "chat", Language: "fr", TermType: RdfLiteral},
	},

	`<http://example.org/graph1> <http://example.org/resource31> <http://example.org/property> "chat"@en .`: {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource31", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "chat", Language: "en", TermType: RdfLiteral},
	},

	"# this is a comment \n<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource1", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "http://example.org/resource2", TermType: RdfIri},
	},

	"# this is a comment \n   # another comment \n<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource1", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "http://example.org/resource2", TermType: RdfIri},
	},

	"<http://example.org/graph1> <http://example.org/resource7> <http://example.org/property> \"typed literal\"^^<http://example.org/DataType1> .": {
		G: RdfTerm{Value: "http://example.org/graph1", TermType: RdfIri},
		S: RdfTerm{Value: "http://example.org/resource7", TermType: RdfIri},
		P: RdfTerm{Value: "http://example.org/property", TermType: RdfIri},
		O: RdfTerm{Value: "typed literal", DataType: "http://example.org/DataType1", TermType: RdfLiteral},
	},
}

var negativeCases = map[string]error{
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ":   ErrUnterminatedQuad,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ,":  ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ..": ErrUnexpectedCharacter,
	"<http://example.org/graph1> http://example.org/resource1> <http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1 <http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1><http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property><http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> http://example.org/property> <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property <http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> http://example.org/resource2>.":    ErrUnexpectedCharacter,
	"<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2.":    ErrUnexpectedEOF,
	"<http://example.org/graph1> <http://example.org/resource1> \n<http://example.org/property> <http://example.org/resource2>.": ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:foo\n <http://example.org/property> <http://example.org/resource2>.":                          ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:0abc <http://example.org/property> <http://example.org/resource2>.":                           ErrUnexpectedCharacter,
	"<http://example.org/graph1> _abc <http://example.org/property> <http://example.org/resource2>.":                             ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:a-bc <http://example.org/property> <http://example.org/resource2>.":                           ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc<http://example.org/property> <http://example.org/resource2>.":                             ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"@ .":                                                 ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^ .":                                                 ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^^< .":                                               ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^^<> .":                                              ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc <> _:abc .":                                                                               ErrUnexpectedCharacter,
	"<http://example.org/graph1> _:abc < > _:abc .":                                                                              ErrUnexpectedCharacter,
}

func TestRead(t *testing.T) {
	for nquad, expected := range testCases {
		t.Run("", func(t *testing.T) {
			r := NewReader(strings.NewReader(nquad))
			ok := r.Next()
			if !ok {
				t.Errorf("Expected %s but got error %s", expected, r.Err())
				return
			}

			if r.Quad() != expected {
				t.Errorf("Expected %s but got %s", expected, r.Quad())
			}
		})
	}
}

func TestReadMultiple(t *testing.T) {
	var nquads bytes.Buffer
	var quads []Quad

	for nquad, quad := range testCases {
		nquads.WriteString(nquad)
		nquads.WriteRune('\n')
		quads = append(quads, quad)
	}

	count := 0
	r := NewReader(strings.NewReader(nquads.String()))

	for r.Next() {
		quad := r.Quad()
		if quad != quads[count] {
			t.Errorf("Expected %s but got %s", quads[count], quad)
			break
		}

		count++
	}

	if r.Err() != nil {
		t.Errorf("Got unexpected error %v", r.Err())
		return
	}

	if count != len(quads) {
		t.Errorf("Expected %d but only parsed %d quads", len(quads), count)

	}

}

func TestReadErrors(t *testing.T) {

	for nquad, expected := range negativeCases {
		t.Run("", func(t *testing.T) {
			r := NewReader(strings.NewReader(nquad))
			r.Next()
			err := r.Err()
			if err == nil {
				t.Errorf("Expected %s for %s but no error reported", expected, nquad)
			} else if err.(*ParseError).Err != expected {
				t.Errorf("Expected %s for %s but got error %s", expected, nquad, err.(*ParseError).Err)
			}
		})
	}
}
