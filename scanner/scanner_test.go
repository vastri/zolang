// Copyright 2016 Vastri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"testing"

	"github.com/vastri/zolang/token"
)

const (
	special = iota
	literal
	operator
)

func tokenclass(tok token.Token) int {
	switch {
	case tok.IsLiteral():
		return literal
	case tok.IsOperator():
		return operator
	}
	return special
}

type elt struct {
	tok   token.Token
	lit   string
	class int
}

var tokens = [...]elt{
	// Special tokens.
	{token.COMMENT, "/* a comment */", special},
	{token.COMMENT, "// a comment \n", special},
	{token.COMMENT, "/*\r*/", special},
	{token.COMMENT, "//\r\n", special},

	// Identifiers and basic type literals.
	{token.IDENT, "foobar", literal},
	{token.IDENT, "a۰۱۸", literal},
	{token.IDENT, "foo६४", literal},
	{token.IDENT, "bar９８７６", literal},
	{token.IDENT, "ŝ", literal},
	{token.IDENT, "ŝfoo", literal},
	{token.BOOL, "true", literal},
	{token.BOOL, "false", literal},
	{token.INT, "0", literal},
	{token.INT, "1", literal},
	{token.INT, "123456789012345678890", literal},
	{token.INT, "01234567", literal},
	{token.INT, "0xcafebabe", literal},
	{token.FLOAT, "0.", literal},
	{token.FLOAT, ".0", literal},
	{token.FLOAT, "3.14159265", literal},
	{token.FLOAT, "1e0", literal},
	{token.FLOAT, "1e+100", literal},
	{token.FLOAT, "1e-100", literal},
	{token.FLOAT, "2.71828e-1000", literal},
	{token.STRING, "\"\"", literal},
	{token.STRING, "\"a\"", literal},
	{token.STRING, "\"foobar\"", literal},
	{token.STRING, "\"${v}\"", literal},
	{token.STRING, "\"foo${v}bar\"", literal},
	{token.RAWSTRING, "''", literal},
	{token.RAWSTRING, "'a'", literal},
	{token.RAWSTRING, "'foobar'", literal},
	{token.RAWSTRING, "'${v}'", literal},
	{token.RAWSTRING, "'foo${v}bar'", literal},

	// Operators and delimiters
	{token.ADD, "+", operator},
	{token.SUB, "-", operator},
	{token.MUL, "*", operator},
	{token.QUO, "/", operator},
	{token.REM, "%", operator},

	{token.AND, "&&", operator},
	{token.OR, "||", operator},

	{token.EQL, "==", operator},
	{token.LSS, "<", operator},
	{token.GTR, ">", operator},
	{token.ASSIGN, "=", operator},
	{token.NOT, "!", operator},

	{token.NEQ, "!=", operator},
	{token.LEQ, "<=", operator},
	{token.GEQ, ">=", operator},

	{token.LPAREN, "(", operator},
	{token.LBRACK, "[", operator},
	{token.LBRACE, "{", operator},
	{token.COMMA, ",", operator},
	{token.PERIOD, ".", operator},

	{token.RPAREN, ")", operator},
	{token.RBRACK, "]", operator},
	{token.RBRACE, "}", operator},
	{token.COLON, ":", operator},
}

const whitespace = "  \t  \n\n\n"

var source = func() []byte {
	var src []byte
	for _, t := range tokens {
		src = append(src, t.lit...)
		src = append(src, whitespace...)
	}
	return src
}()

func newlineCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func checkPos(t *testing.T, fset *token.FileSet, lit string, p token.Pos, expected token.Position) {
	pos := fset.Position(p)
	if pos.Filename != expected.Filename {
		t.Errorf("bad filename for %q: got %s, expected %s", lit, pos.Filename, expected.Filename)
	}
	if pos.Offset != expected.Offset {
		t.Errorf("bad position for %q: got %d, expected %d", lit, pos.Offset, expected.Offset)
	}
	if pos.Line != expected.Line {
		t.Errorf("bad line for %q: got %d, expected %d", lit, pos.Line, expected.Line)
	}
	if pos.Column != expected.Column {
		t.Errorf("bad column for %q: got %d, expected %d", lit, pos.Column, expected.Column)
	}
}

// TestScan verifies that calling Scan() provides the correct results.
func TestScan(t *testing.T) {
	fset := token.NewFileSet()

	whitespace_linecount := newlineCount(whitespace)

	// Error handler.
	eh := func(_ token.Position, msg string) {
		t.Errorf("error handler called (msg = %s)", msg)
	}

	// Verify scan.
	var s Scanner
	s.Init(fset.AddFile("", fset.Base(), len(source)), source, eh)

	// Set up expected position.
	epos := token.Position{
		Filename: "",
		Offset:   0,
		Line:     1,
		Column:   1,
	}

	index := 0
	for {
		pos, tok, lit := s.Scan()

		// Check position.
		if tok == token.EOF {
			// Correction for EOF.
			epos.Line = newlineCount(string(source))
			epos.Column = 2
		}
		checkPos(t, fset, lit, pos, epos)

		// Check token.
		e := elt{token.EOF, "", special}
		if index < len(tokens) {
			e = tokens[index]
			index++
		}
		if tok != e.tok {
			t.Errorf("bad token for %q: got %s, expected %s", e.lit, tok, e.tok)
		}

		// Check token class.
		if tokenclass(tok) != e.class {
			t.Errorf("bad class for %q: got %d, expected %d", e.lit, tokenclass(tok), e.class)
		}

		// Check literal.
		elit := ""
		if tok.IsLiteral() {
			elit = e.lit
		}
		if lit != elit {
			t.Errorf("bad literal for %q: got %q, expected %q", e.lit, lit, elit)
		}

		if tok == token.EOF {
			break
		}

		// Update position.
		epos.Offset += len(e.lit) + len(whitespace)
		epos.Line += newlineCount(e.lit) + whitespace_linecount
	}

	if s.ErrorCount != 0 {
		t.Errorf("found %d errors", s.ErrorCount)
	}
}

func BenchmarkScan(b *testing.B) {
	b.StopTimer()
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(source))
	var s Scanner
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Init(file, source, nil)
		for {
			_, tok, _ := s.Scan()
			if tok == token.EOF {
				break
			}
		}
	}
}
