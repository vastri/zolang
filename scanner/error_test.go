// Copyright 2016 Vastri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scanner

import (
	"os"
	"testing"

	"github.com/vastri/zolang/token"
)

func testError(t *testing.T) {
	const src = "@\n@ @\n"

	fset := token.NewFileSet()

	var list ErrorList
	eh := func(pos token.Position, msg string) { list.Add(pos, msg) }

	var s Scanner
	s.Init(fset.AddFile("File1", fset.Base(), len(src)), []byte(src), eh)

	for {
		if _, tok, _ := s.Scan(); tok == token.EOF {
			break
		}
	}

	if len(list) != s.ErrorCount {
		t.Errorf("found %d errors, expected %d", len(list), s.ErrorCount)
	}

	if len(list) != 3 {
		t.Errorf("found %d raw errors, expected 3", len(list))
		PrintError(os.Stderr, list)
	}

	list.Sort()
	if len(list) != 3 {
		t.Errorf("found %d sorted errors, expected 3", len(list))
		PrintError(os.Stderr, list)
	}

	list.RemoveMultiples()
	if len(list) != 2 {
		t.Errorf("found %d lines with errors, expected 2", len(list))
		PrintError(os.Stderr, list)
	}
}

type errorCollector struct {
	cnt int            // number of errors encountered
	msg string         // last error message encountered
	pos token.Position // last error position encountered
}

func checkError(t *testing.T, fset *token.FileSet, src string, tok token.Token, pos int, lit, err string) {
	var s Scanner
	var h errorCollector
	eh := func(pos token.Position, msg string) {
		h.cnt++
		h.msg = msg
		h.pos = pos
	}
	s.Init(fset.AddFile("", fset.Base(), len(src)), []byte(src), eh)
	_, tok0, lit0 := s.Scan()
	if tok0 != tok {
		t.Errorf("%q: got %s, expected %s", src, tok0, tok)
	}
	if tok0 != token.ILLEGAL && lit0 != lit {
		t.Errorf("%q: got literal %q, expected %q", src, lit0, lit)
	}
	cnt := 0
	if err != "" {
		cnt = 1
	}
	if h.cnt != cnt {
		t.Errorf("%q: got cnt %d, expected %d", src, h.cnt, cnt)
	}
	if h.msg != err {
		t.Errorf("%q: got msg %q, expected %q", src, h.msg, err)
	}
	if h.pos.Offset != pos {
		t.Errorf("%q: got offset %d, expected %d", src, h.pos.Offset, pos)
	}
}

var errors = []struct {
	src string
	tok token.Token
	pos int
	lit string
	err string
}{
	{"\a", token.ILLEGAL, 0, "", "illegal character U+0007"},
	{`#`, token.ILLEGAL, 0, "", "illegal character U+0023 '#'"},
	{`…`, token.ILLEGAL, 0, "", "illegal character U+2026 '…'"},
	{`' '`, token.RAWSTRING, 0, `' '`, ""},
	{`''`, token.RAWSTRING, 0, `''`, ""},
	{`'12'`, token.RAWSTRING, 0, `'12'`, ""},
	{`'123'`, token.RAWSTRING, 0, `'123'`, ""},
	{`'\0'`, token.RAWSTRING, 3, `'\0'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\07'`, token.RAWSTRING, 4, `'\07'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\8'`, token.RAWSTRING, 2, `'\8'`, "unknown escape sequence"},
	{`'\08'`, token.RAWSTRING, 3, `'\08'`, "illegal character U+0038 '8' in escape sequence"},
	{`'\x'`, token.RAWSTRING, 3, `'\x'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\x0'`, token.RAWSTRING, 4, `'\x0'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\x0g'`, token.RAWSTRING, 4, `'\x0g'`, "illegal character U+0067 'g' in escape sequence"},
	{`'\u'`, token.RAWSTRING, 3, `'\u'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\u0'`, token.RAWSTRING, 4, `'\u0'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\u00'`, token.RAWSTRING, 5, `'\u00'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\u000'`, token.RAWSTRING, 6, `'\u000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\u0000'`, token.RAWSTRING, 0, `'\u0000'`, ""},
	{`'\U'`, token.RAWSTRING, 3, `'\U'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U0'`, token.RAWSTRING, 4, `'\U0'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U00'`, token.RAWSTRING, 5, `'\U00'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U000'`, token.RAWSTRING, 6, `'\U000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U0000'`, token.RAWSTRING, 7, `'\U0000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U00000'`, token.RAWSTRING, 8, `'\U00000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U000000'`, token.RAWSTRING, 9, `'\U000000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U0000000'`, token.RAWSTRING, 10, `'\U0000000'`, "illegal character U+0027 ''' in escape sequence"},
	{`'\U00000000'`, token.RAWSTRING, 0, `'\U00000000'`, ""},
	{`'\Uffffffff'`, token.RAWSTRING, 2, `'\Uffffffff'`, "escape sequence is invalid Unicode code point"},
	{`'`, token.RAWSTRING, 0, `'`, "string literal not terminated"},
	{`'\'`, token.RAWSTRING, 0, `'\'`, "string literal not terminated"},
	{"'\n", token.RAWSTRING, 0, "'", "string literal not terminated"},
	{"'\n   ", token.RAWSTRING, 0, "'", "string literal not terminated"},
	{`""`, token.STRING, 0, `""`, ""},
	{`"abc`, token.STRING, 0, `"abc`, "string literal not terminated"},
	{"\"abc\n", token.STRING, 0, `"abc`, "string literal not terminated"},
	{"\"abc\n   ", token.STRING, 0, `"abc`, "string literal not terminated"},
	{"/**/ /*", token.COMMENT, 0, "", ""},
	{"/*", token.COMMENT, 0, "", ""},
	{"077", token.INT, 0, "077", ""},
	{"078.", token.FLOAT, 0, "078.", ""},
	{"07801234567.", token.FLOAT, 0, "07801234567.", ""},
	{"078e0", token.FLOAT, 0, "078e0", ""},
	{"078", token.INT, 0, "078", "illegal octal number"},
	{"07800000009", token.INT, 0, "07800000009", "illegal octal number"},
	{"0x", token.INT, 0, "0x", "illegal hexadecimal number"},
	{"0X", token.INT, 0, "0X", "illegal hexadecimal number"},
	{"\"abc\x00def\"", token.STRING, 4, "\"abc\x00def\"", "illegal character NUL"},
	{"\"abc\x80def\"", token.STRING, 4, "\"abc\x80def\"", "illegal UTF-8 encoding"},
	{"\ufeff\ufeff", token.ILLEGAL, 3, "\ufeff\ufeff", "illegal byte order mark"},                        // only first BOM is ignored
	{"//\ufeff", token.COMMENT, 2, "", "illegal byte order mark"},                                        // only first BOM is ignored
	{`"` + "abc\ufeffdef" + `"`, token.STRING, 4, `"` + "abc\ufeffdef" + `"`, "illegal byte order mark"}, // only first BOM is ignored
}

func TestScanErrors(t *testing.T) {
	fset := token.NewFileSet()
	for _, e := range errors {
		checkError(t, fset, e.src, e.tok, e.pos, e.lit, e.err)
	}
}
