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
