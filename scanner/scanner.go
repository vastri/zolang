// Copyright 2016 Vastri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scanner implements a scanner for zolang.
package scanner

import (
	"fmt"
	"path/filepath"
	"unicode"
	"unicode/utf8"

	"github.com/vastri/zolang/token"
)

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
//
type ErrorHandler func(pos token.Position, msg string)

// A Scanner holds the scanner's internal state while processing
// a given text. It can be allocated as part of another data
// structure but must be initialized via Init before use.
//
type Scanner struct {
	// Immutable state.
	file *token.File  // source file handle
	dir  string       // directory portion of file.Name()
	src  []byte       // source
	err  ErrorHandler // error reporting; or nil

	// Scanning state.
	ch       rune // current character
	offset   int  // character offset
	rdOffset int  // reading offset (position after current character)

	// Public state - ok to modify.
	ErrorCount int // number of errors encountered
}

const bom = 0xFEFF // byte order mark, only permitted as very first character

// next reads the next unicode char into s.ch.
// s.ch < 0 means end-of-file.
//
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// Not ASCII.
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		s.ch = -1 // eof
	}
}

// Init prepares the scanner s to tokenize the text src by setting the
// scanner at the beginning of src. The scanner uses the file set file
// for position information and it adds line information for each line.
// It is ok to re-use the same file when re-scanning the same file as
// line information which is already present is ignored. Init causes a
// panic if the file size does not match the src size.
//
// Calls to Scan will invoke the error handler err if they encounter a
// syntax error and err is not nil. Also, for each error encountered,
// the Scanner field ErrorCount is incremented by one.
//
// Note that Init may call err if there is an error in the fisrt character
// of the file.
//
func (s *Scanner) Init(file *token.File, src []byte, err ErrorHandler) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.dir, _ = filepath.Split(file.Name())
	s.src = src
	s.err = err

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0
	s.ErrorCount = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
	s.ErrorCount++
}

func (s *Scanner) scanComment() {
	// Initial '/' already consumed; s.ch == '/' || s.ch == '*'.
	if s.ch == '/' {
		// Single-line comment.
		s.next()
		for s.ch != '\n' && s.ch >= 0 {
			s.next()
		}
	} else {
		// Multi-line comment.
		s.next()
		for s.ch >= 0 {
			ch := s.ch
			s.next()
			if ch == '*' && s.ch == '/' {
				s.next()
				break
			}
		}
	}
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || ch >= utf8.RuneSelf && unicode.IsDigit(ch)
}

func (s *Scanner) scanIdentifier() string {
	offs := s.offset
	for isLetter(s.ch) || isDigit(s.ch) {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func digitVal(ch rune) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'a' <= ch && ch <= 'f':
		return int(ch - 'a' + 10)
	case 'A' <= ch && ch <= 'F':
		return int(ch - 'A' + 10)
	}
	return 16 // larger than any legal digit val
}

func (s *Scanner) scanMantissa(base int) {
	for digitVal(s.ch) < base {
		s.next()
	}
}

func (s *Scanner) scanNumber(seenDecimalPoint bool) (token.Token, string) {
	// digitVal(s.ch) < 10.
	offs := s.offset
	tok := token.INT

	if seenDecimalPoint {
		offs--
		tok = token.FLOAT
		s.scanMantissa(10)
		goto exponent
	}

	if s.ch == '0' {
		// int or float.
		offs := s.offset
		s.next()
		if s.ch == 'x' || s.ch == 'X' {
			// Hexadecimal int.
			s.next()
			s.scanMantissa(16)
			if s.offset-offs <= 2 {
				// Only scanned "0x" or "0X".
				s.error(offs, "illegal hexadecimal number")
			}
		} else {
			// Octal int or float.
			seenDecimalPoint := false
			s.scanMantissa(8)
			if s.ch == '8' || s.ch == '9' {
				// Illegal octal int or float.
				seenDecimalPoint = true
				s.scanMantissa(10)
			}
			if s.ch == '.' || s.ch == 'e' || s.ch == 'E' {
				goto fraction
			}
			// Octal int.
			if seenDecimalPoint {
				s.error(offs, "illegal octal number")
			}
		}
		goto exit
	}

	// Decimal int or float.
	s.scanMantissa(10)

fraction:
	if s.ch == '.' {
		tok = token.FLOAT
		s.next()
		s.scanMantissa(10)
	}

exponent:
	if s.ch == 'e' || s.ch == 'E' {
		tok = token.FLOAT
		s.next()
		if s.ch == '-' || s.ch == '+' {
			s.next()
		}
		s.scanMantissa(10)
	}

exit:
	return tok, string(s.src[offs:s.offset])
}

// scanEscape parses an escape sequence where rune is the accepted
// escaped quote. In case of a syntax error, it stops at the offending
// character (without consuming it) and returns false. Otherwise
// it returns true.
//
func (s *Scanner) scanEscape(quote rune) bool {
	offs := s.offset

	var n int
	var base, max uint32
	switch s.ch {
	case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', quote:
		s.next()
		return true
	case '0', '1', '2', '3', '4', '5', '6', '7':
		n, base, max = 3, 8, 255
	case 'x':
		s.next()
		n, base, max = 2, 16, 255
	case 'u':
		s.next()
		n, base, max = 4, 16, unicode.MaxRune
	case 'U':
		s.next()
		n, base, max = 8, 16, unicode.MaxRune
	default:
		msg := "unknown escape sequence"
		if s.ch < 0 {
			msg = "escape sequence not terminated"
		}
		s.error(offs, msg)
		return false
	}

	var x uint32
	for n > 0 {
		d := uint32(digitVal(s.ch))
		if d >= base {
			msg := fmt.Sprintf("illegal character %#U in escape sequence", s.ch)
			if s.ch < 0 {
				msg = "escape sequence not terminated"
			}
			s.error(s.offset, msg)
			return false
		}
		x = x*base + d
		s.next()
		n--
	}

	if x > max || 0xD800 <= x && x < 0xE000 {
		s.error(offs, "escape sequence is invalid Unicode code point")
		return false
	}

	return true
}

func (s *Scanner) scanString(quote rune) string {
	// Quote opening already consumed.
	offs := s.offset - 1

	for {
		ch := s.ch
		if ch == '\n' || ch < 0 {
			s.error(offs, "string literal not terminated")
			break
		}
		s.next()
		if ch == quote {
			break
		}
		if ch == '\\' {
			s.scanEscape(quote)
		}
	}

	return string(s.src[offs:s.offset])
}

func (s *Scanner) skipWhiteSpace() {
	for s.ch == ' ' || s.ch == '\t' || s.ch == '\n' || s.ch == '\r' {
		s.next()
	}
}

// Scan scans the next token and returns the token position, the token,
// and its literal string if applicable. The source end is indicated by
// token.EOF.
//
// If the returned token is literal (token.IDENT, token.BOOL, token.INT,
// token.FLOAT, token.STRING) or token.COMMENT, the literal string has
// the corresponding value.
//
// If the returned token is token.ILLEGAL, the literal string is the
// offending character.
//
// In all other cases, Scan returns an empty literal string.
//
// For more tolerant parsing, Scan will return a valid token if
// possible even if a syntax error was encountered. Thus, even
// if the resulting token sequence contains no illegal token,
// a client may not assume that no error occurred. Instead it
// must check the scanner's ErrorCount or the number of calls
// of the error handler, if there was one installed.
//
// Scan adds line information to the file added to the file
// set with Init. Token positions are relative to that file
// and thus relative to the file set.
//
func (s *Scanner) Scan() (pos token.Pos, tok token.Token, lit string) {
	s.skipWhiteSpace()

	// Current token start.
	pos = s.file.Pos(s.offset)

	// Determine token value.
	switch ch := s.ch; {
	case isLetter(ch):
		lit = s.scanIdentifier()
		if lit == "true" || lit == "false" {
			tok = token.BOOL
		} else {
			tok = token.IDENT
		}
	case '0' <= ch && ch <= '9':
		tok, lit = s.scanNumber(false)
	default:
		s.next() // always make progress
		switch ch {
		case -1:
			tok = token.EOF
		case '"':
			tok = token.STRING
			lit = s.scanString('"')
		case '\'':
			tok = token.RAWSTRING
			lit = s.scanString('\'')
		case ':':
			tok = token.COLON
		case '.':
			if '0' <= s.ch && s.ch <= '9' {
				tok, lit = s.scanNumber(true)
			} else {
				tok = token.PERIOD
			}
		case ',':
			tok = token.COMMA
		case '(':
			tok = token.LPAREN
		case ')':
			tok = token.RPAREN
		case '[':
			tok = token.LBRACK
		case ']':
			tok = token.RBRACK
		case '{':
			tok = token.LBRACE
		case '}':
			tok = token.RBRACE
		case '+':
			tok = token.ADD
		case '-':
			tok = token.SUB
		case '*':
			tok = token.MUL
		case '/':
			if s.ch == '/' || s.ch == '*' {
				tok = token.COMMENT
				s.scanComment()
			} else {
				tok = token.QUO
			}
		case '%':
			tok = token.REM
		case '<':
			if s.ch == '=' {
				s.next()
				tok = token.LEQ
			} else {
				tok = token.LSS
			}
		case '>':
			if s.ch == '=' {
				s.next()
				tok = token.GEQ
			} else {
				tok = token.GTR
			}
		case '=':
			if s.ch == '=' {
				s.next()
				tok = token.EQL
			} else {
				tok = token.ASSIGN
			}
		case '!':
			if s.ch == '=' {
				s.next()
				tok = token.NEQ
			} else {
				tok = token.NOT
			}
		case '&':
			if s.ch == '&' {
				s.next()
				tok = token.AND
			} else {
				s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
				tok = token.ILLEGAL
				lit = string(ch)
			}
		case '|':
			if s.ch == '|' {
				s.next()
				tok = token.OR
			} else {
				s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
				tok = token.ILLEGAL
				lit = string(ch)
			}
		default:
			// next reports unexpected BOMs - don't repeat.
			if ch != bom {
				s.error(s.file.Offset(pos), fmt.Sprintf("illegal character %#U", ch))
			}
			tok = token.ILLEGAL
			lit = string(ch)
		}
	}

	return
}
