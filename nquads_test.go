/*
  This is free and unencumbered software released into the public domain. For more
  information, see <http://unlicense.org/> or the accompanying UNLICENSE file.
*/

package nquads

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iand/gordf"
)

var parseCases = []struct {
	name     string
	filename string
	inline   string
	quads    []Quad
}{
	{
		name:   "",
		inline: "<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource1"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},

	{
		name:   "",
		inline: "_:anon <http://example.org/property> <http://example.org/resource2> <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.Blank("anon"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: "<http://example.org/resource1> <http://example.org/property> _:anon <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource1"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Blank("anon"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: "	 <http://example.org/resource3> 	 <http://example.org/property>	 <http://example.org/resource2> 	   <http://example.org/graph1>    .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource3"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: "<http://example.org/resource7> <http://example.org/property> \"simple literal\" <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource7"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("simple literal"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: `<http://example.org/resource8> <http://example.org/property> "backslash:\\" <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource8"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("backslash:\\"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: `<http://example.org/resource9> <http://example.org/property> "dquote:\"" <http://example.org/graph1>.`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource9"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("dquote:\""),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "",
		inline: `<http://example.org/resource10> <http://example.org/property> "newline:\n" <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource10"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("newline:\n"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name: "",

		inline: `<http://example.org/resource11> <http://example.org/property> "return\r" <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource11"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("return\r"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name: "",

		inline: `<http://example.org/resource12> <http://example.org/property> "tab:\t" <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource12"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("tab:\t"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name: "",

		inline: `<http://example.org/resource16> <http://example.org/property> "\u00E9" <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource16"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.Literal("\u00E9"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "lang-literal-01",
		inline: `<http://example.org/resource30> <http://example.org/property> "chat"@fr <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource30"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.LiteralWithLanguage("chat", "fr"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "lang-literal-02",
		inline: `<http://example.org/resource31> <http://example.org/property> "chat"@en-UK <http://example.org/graph1> .`,
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource31"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.LiteralWithLanguage("chat", "en-UK"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "leading-comment",
		inline: "# this is a comment \n<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource1"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "trailing-comment",
		inline: "<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> <http://example.org/graph1> .# this is a comment",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource1"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "leading-multiple-comments",
		inline: "# this is a comment \n   # another comment \n<http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> <http://example.org/graph1> .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource1"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.IRI("http://example.org/resource2"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},

	{
		name:   "",
		inline: "<http://example.org/resource7> <http://example.org/property> \"typed literal\"^^<http://example.org/DataType1> <http://example.org/graph1>  .",
		quads: []Quad{
			{
				S: rdf.IRI("http://example.org/resource7"),
				P: rdf.IRI("http://example.org/property"),
				O: rdf.LiteralWithDatatype("typed literal", "http://example.org/DataType1"),
				G: rdf.IRI("http://example.org/graph1"),
			},
		},
	},
	{
		name:   "minimal-whitespace-triple",
		inline: "<http://example/s><http://example/p>_:o.",
		quads: []Quad{
			{
				S: rdf.IRI("http://example/s"),
				P: rdf.IRI("http://example/p"),
				O: rdf.Blank("o"),
				G: rdf.Term{Kind: rdf.UnknownTerm},
			},
		},
	},
	{
		name:   "minimal-whitespace-quad",
		inline: "<http://example/s><http://example/p><http://example/o><http://example/g>.",
		quads: []Quad{
			{
				S: rdf.IRI("http://example/s"),
				P: rdf.IRI("http://example/p"),
				O: rdf.IRI("http://example/o"),
				G: rdf.IRI("http://example/g"),
			},
		},
	},
}

var positiveSyntaxCases = []struct {
	name     string
	filename string
	inline   string
}{
	{
		name:     "w3c-comment_following_triple",
		filename: "testdata/w3c-test-suite/comment_following_triple.nq",
	},
	{
		name:     "w3c-langtagged_string",
		filename: "testdata/w3c-test-suite/langtagged_string.nq",
	},
	{
		name:     "w3c-lantag_with_subtag",
		filename: "testdata/w3c-test-suite/lantag_with_subtag.nq",
	},
	{
		name:     "w3c-literal_all_controls",
		filename: "testdata/w3c-test-suite/literal_all_controls.nq",
	},
	{
		name:     "w3c-literal_all_punctuation",
		filename: "testdata/w3c-test-suite/literal_all_punctuation.nq",
	},
	{
		name:     "w3c-literal_ascii_boundaries",
		filename: "testdata/w3c-test-suite/literal_ascii_boundaries.nq",
	},
	{
		name:     "w3c-literal_false",
		filename: "testdata/w3c-test-suite/literal_false.nq",
	},
	{
		name:     "w3c-literal",
		filename: "testdata/w3c-test-suite/literal.nq",
	},
	{
		name:     "w3c-literal_true",
		filename: "testdata/w3c-test-suite/literal_true.nq",
	},
	{
		name:     "w3c-literal_with_2_dquotes",
		filename: "testdata/w3c-test-suite/literal_with_2_dquotes.nq",
	},
	{
		name:     "w3c-literal_with_2_squotes",
		filename: "testdata/w3c-test-suite/literal_with_2_squotes.nq",
	},
	{
		name:     "w3c-literal_with_BACKSPACE",
		filename: "testdata/w3c-test-suite/literal_with_BACKSPACE.nq",
	},
	{
		name:     "w3c-literal_with_CARRIAGE_RETURN",
		filename: "testdata/w3c-test-suite/literal_with_CARRIAGE_RETURN.nq",
	},
	{
		name:     "w3c-literal_with_CHARACTER_TABULATION",
		filename: "testdata/w3c-test-suite/literal_with_CHARACTER_TABULATION.nq",
	},
	{
		name:     "w3c-literal_with_dquote",
		filename: "testdata/w3c-test-suite/literal_with_dquote.nq",
	},
	{
		name:     "w3c-literal_with_FORM_FEED",
		filename: "testdata/w3c-test-suite/literal_with_FORM_FEED.nq",
	},
	{
		name:     "w3c-literal_with_LINE_FEED",
		filename: "testdata/w3c-test-suite/literal_with_LINE_FEED.nq",
	},
	{
		name:     "w3c-literal_with_numeric_escape4",
		filename: "testdata/w3c-test-suite/literal_with_numeric_escape4.nq",
	},
	{
		name:     "w3c-literal_with_numeric_escape8",
		filename: "testdata/w3c-test-suite/literal_with_numeric_escape8.nq",
	},
	{
		name:     "w3c-literal_with_REVERSE_SOLIDUS2",
		filename: "testdata/w3c-test-suite/literal_with_REVERSE_SOLIDUS2.nq",
	},
	{
		name:     "w3c-literal_with_REVERSE_SOLIDUS",
		filename: "testdata/w3c-test-suite/literal_with_REVERSE_SOLIDUS.nq",
	},
	{
		name:     "w3c-literal_with_squote",
		filename: "testdata/w3c-test-suite/literal_with_squote.nq",
	},
	{
		name:     "w3c-literal_with_UTF8_boundaries",
		filename: "testdata/w3c-test-suite/literal_with_UTF8_boundaries.nq",
	},
	{
		name:     "w3c-minimal_whitespace",
		filename: "testdata/w3c-test-suite/minimal_whitespace.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-01",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-01.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-02",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-02.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-03",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-03.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-04",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-04.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-05",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-05.nq",
	},
	{
		name:     "w3c-nq-syntax-bnode-06",
		filename: "testdata/w3c-test-suite/nq-syntax-bnode-06.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-01",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-01.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-02",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-02.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-03",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-03.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-04",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-04.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-05",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-05.nq",
	},
	{
		name:     "w3c-nq-syntax-uri-06",
		filename: "testdata/w3c-test-suite/nq-syntax-uri-06.nq",
	},
	{
		name:     "w3c-nt-syntax-bnode-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bnode-01.nq",
	},
	{
		name:     "w3c-nt-syntax-bnode-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bnode-02.nq",
	},
	{
		name:     "w3c-nt-syntax-bnode-03",
		filename: "testdata/w3c-test-suite/nt-syntax-bnode-03.nq",
	},
	{
		name:     "w3c-nt-syntax-datatypes-01",
		filename: "testdata/w3c-test-suite/nt-syntax-datatypes-01.nq",
	},
	{
		name:     "w3c-nt-syntax-datatypes-02",
		filename: "testdata/w3c-test-suite/nt-syntax-datatypes-02.nq",
	},
	{
		name:     "w3c-nt-syntax-file-01",
		filename: "testdata/w3c-test-suite/nt-syntax-file-01.nq",
	},
	{
		name:     "w3c-nt-syntax-file-02",
		filename: "testdata/w3c-test-suite/nt-syntax-file-02.nq",
	},
	{
		name:     "w3c-nt-syntax-file-03",
		filename: "testdata/w3c-test-suite/nt-syntax-file-03.nq",
	},
	{
		name:     "w3c-nt-syntax-str-esc-01",
		filename: "testdata/w3c-test-suite/nt-syntax-str-esc-01.nq",
	},
	{
		name:     "w3c-nt-syntax-str-esc-02",
		filename: "testdata/w3c-test-suite/nt-syntax-str-esc-02.nq",
	},
	{
		name:     "w3c-nt-syntax-str-esc-03",
		filename: "testdata/w3c-test-suite/nt-syntax-str-esc-03.nq",
	},
	{
		name:     "w3c-nt-syntax-string-01",
		filename: "testdata/w3c-test-suite/nt-syntax-string-01.nq",
	},
	{
		name:     "w3c-nt-syntax-string-02",
		filename: "testdata/w3c-test-suite/nt-syntax-string-02.nq",
	},
	{
		name:     "w3c-nt-syntax-string-03",
		filename: "testdata/w3c-test-suite/nt-syntax-string-03.nq",
	},
	{
		name:     "w3c-nt-syntax-subm-01",
		filename: "testdata/w3c-test-suite/nt-syntax-subm-01.nq",
	},
	{
		name:     "w3c-nt-syntax-uri-01",
		filename: "testdata/w3c-test-suite/nt-syntax-uri-01.nq",
	},
	{
		name:     "w3c-nt-syntax-uri-02",
		filename: "testdata/w3c-test-suite/nt-syntax-uri-02.nq",
	},
	{
		name:     "w3c-nt-syntax-uri-03",
		filename: "testdata/w3c-test-suite/nt-syntax-uri-03.nq",
	},
	{
		name:     "w3c-nt-syntax-uri-04",
		filename: "testdata/w3c-test-suite/nt-syntax-uri-04.nq",
	},
}

var negativeSyntaxCases = []struct {
	name     string
	filename string
	inline   string
	err      error
}{
	{
		name:   "no-terminating-dot",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ",
		err:    ErrUnterminatedQuad,
	},
	{
		name:   "wrong-terminating-character",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ,",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "noncomment-after-dot",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2> ..",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> http://example.org/resource1> <http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> <http://example.org/resource1 <http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> <http://example.org/resource1> http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "unterminated-iri",
		inline: "<http://example.org/graph1> <http://example.org/resource1> <http://example.org/property> <http://example.org/resource2.",
		err:    ErrUnexpectedEOF,
	},
	{
		name:   "newline-in-quad",
		inline: "<http://example.org/graph1> <http://example.org/resource1> \n<http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:foo\n <http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _abc <http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc<http://example.org/property> <http://example.org/resource2>.",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"@ .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^ .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^^< .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc <http://example.org/property> \"foo\"^^<> .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc <> _:abc .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:   "",
		inline: "<http://example.org/graph1> _:abc < > _:abc .",
		err:    ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nq-syntax-bad-literal-01",
		filename: "testdata/w3c-test-suite/nq-syntax-bad-literal-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nq-syntax-bad-literal-02",
		filename: "testdata/w3c-test-suite/nq-syntax-bad-literal-02.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nq-syntax-bad-literal-03",
		filename: "testdata/w3c-test-suite/nq-syntax-bad-literal-03.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nq-syntax-bad-quint-01",
		filename: "testdata/w3c-test-suite/nq-syntax-bad-quint-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-base-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-base-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-esc-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-esc-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-esc-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-esc-02.nq",
		err:      ErrInvalidCodepointExpression,
	},
	{
		name:     "w3c-nt-syntax-bad-esc-03",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-esc-03.nq",
		err:      ErrInvalidCodepointExpression,
	},
	{
		name:     "w3c-nt-syntax-bad-lang-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-esc-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-num-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-num-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-num-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-num-02.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-num-03",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-num-03.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-prefix-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-prefix-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-string-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-01.nq",
		err:      ErrUnexpectedEOF,
	},
	{
		name:     "w3c-nt-syntax-bad-string-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-02.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-string-03",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-03.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-string-04",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-04.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-string-05",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-05.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-string-06",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-06.nq",
		err:      ErrUnexpectedEOF,
	},
	{
		name:     "w3c-nt-syntax-bad-string-07",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-string-07.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-struct-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-struct-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-struct-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-struct-02.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-01",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-01.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-02",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-02.nq",
		err:      ErrInvalidCodepointExpression,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-03",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-03.nq",
		err:      ErrInvalidCodepointExpression,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-04",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-04.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-05",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-05.nq",
		err:      ErrUnexpectedCharacter,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-06",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-06.nq",
		err:      ErrRelativeIRI,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-07",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-07.nq",
		err:      ErrRelativeIRI,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-08",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-08.nq",
		err:      ErrRelativeIRI,
	},
	{
		name:     "w3c-nt-syntax-bad-uri-09",
		filename: "testdata/w3c-test-suite/nt-syntax-bad-uri-09.nq",
		err:      ErrRelativeIRI,
	},
}

func TestParse(t *testing.T) {
	for _, tc := range parseCases {
		t.Run(tc.name, func(t *testing.T) {
			var r io.ReadCloser
			if tc.filename != "" {
				var err error
				r, err = os.Open(tc.filename)
				if err != nil {
					t.Fatalf("failed to open test file %s: %v", tc.filename, err)
				}
			} else if tc.inline != "" {
				r = io.NopCloser(strings.NewReader(tc.inline))
			} else {
				t.Fatal("invalid test case: expecting one of filename or inline to be specified")
			}
			defer r.Close()

			nqr := NewReader(r)
			for i, quad := range tc.quads {
				ok := nqr.Next()
				if !ok {
					t.Fatalf("quad %d: missing (err=%v)", i, nqr.Err())
				}

				if nqr.Quad() != quad {
					t.Errorf("quad %d: got %s, wanted %s", i, nqr.Quad(), quad)
				}
			}

			if nqr.Next() {
				t.Errorf("got additional unexpected quad %s", nqr.Quad())
			}
		})
	}
}

func TestPositiveSyntaxCases(t *testing.T) {
	for _, tc := range positiveSyntaxCases {
		t.Run(tc.name, func(t *testing.T) {
			var r io.ReadCloser
			if tc.filename != "" {
				var err error
				r, err = os.Open(tc.filename)
				if err != nil {
					t.Fatalf("failed to open test file %s: %v", tc.filename, err)
				}
			} else if tc.inline != "" {
				r = io.NopCloser(strings.NewReader(tc.inline))
			} else {
				t.Fatal("invalid test case: expecting one of filename or inline to be specified")
			}
			defer r.Close()

			nqr := NewReader(r)
			nqr.Next()
			err := nqr.Err()
			if err != nil {
				t.Errorf("got %v error reported, wanted no error", err)
			}
		})
	}
}

func TestNegativeSyntaxCases(t *testing.T) {
	for _, tc := range negativeSyntaxCases {
		t.Run(tc.name, func(t *testing.T) {
			var r io.ReadCloser
			if tc.filename != "" {
				var err error
				r, err = os.Open(tc.filename)
				if err != nil {
					t.Fatalf("failed to open test file %s: %v", tc.filename, err)
				}
			} else if tc.inline != "" {
				r = io.NopCloser(strings.NewReader(tc.inline))
			} else {
				t.Fatal("invalid test case: expecting one of filename or inline to be specified")
			}
			defer r.Close()

			nqr := NewReader(r)
			nqr.Next()
			err := nqr.Err()
			if err == nil {
				t.Errorf("no error reported, wanted %v", tc.err)
			} else if !errors.Is(err, tc.err) {
				t.Errorf("got %v error reported, wanted %v", err, tc.err)
			}
		})
	}
}

func TestParseIRI(t *testing.T) {
	testCases := []struct {
		input string
		value string
		err   error
	}{
		{
			input: `http://example/\u00ZZ11>`,
			err:   ErrInvalidCodepointExpression,
		},
		{
			input: `http://example/\U00ZZ1111>`,
			err:   ErrInvalidCodepointExpression,
		},
		{
			input: `http://example/\n>`,
			err:   ErrUnexpectedCharacter,
		},
		{
			input: `http://example/\/>`,
			err:   ErrUnexpectedCharacter,
		},
		{
			input: `s`,
			err:   ErrUnexpectedCharacter,
		},
		{
			input: `http://example.com/foo\u00E9>`,
			value: `http://example.com/fooé`,
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			// Append space to input to act as delimiter
			nqr := NewReader(strings.NewReader(tc.input + " "))

			term, err := nqr.parseIRI()
			switch {
			case tc.err == nil && err != nil:
				t.Fatalf("got unexpected error %q", err)
			case tc.err != nil && err == nil:
				t.Fatalf("got no error, wanted %q", tc.err)
			case !errors.Is(err, tc.err):
				t.Fatalf("got error %q, wanted %q", err, tc.err)
			}

			if term.Value != tc.value {
				t.Errorf("got value %v, wanted %v", term.Value, tc.value)
			}
		})
	}
}

func TestParseBlankNode(t *testing.T) {
	testCases := []struct {
		input string
		value string
		err   error
	}{
		{
			input: ":a", // _:a
			value: "a",
		},
		{
			input: ":1a", // _:1a
			value: "1a",
		},
		{
			input: "a", // _a
			err:   ErrUnexpectedCharacter,
		},
		{
			input: ":_a", // _:_a
			value: "_a",
		},
		{
			input: "::a", // _::a
			value: ":a",
		},
		{
			input: ":a.b", // _:a.b
			value: "a.b",
		},
		{
			input: ":.b", // _:.b
			err:   ErrUnexpectedCharacter,
		},
		{
			input: ":a.", // _:a.
			value: "a",
		},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			// Append space to input to act as delimiter
			nqr := NewReader(strings.NewReader(tc.input + " "))

			term, err := nqr.parseBlankNode()
			switch {
			case tc.err == nil && err != nil:
				t.Fatalf("got unexpected error %q", err)
			case tc.err != nil && err == nil:
				t.Fatalf("got no error, wanted %q", tc.err)
			case !errors.Is(err, tc.err):
				t.Fatalf("got error %q, wanted %q", err, tc.err)
			}

			if term.Value != tc.value {
				t.Errorf("got value %v, wanted %v", term.Value, tc.value)
			}
		})
	}
}
