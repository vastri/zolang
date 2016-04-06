// Copyright 2016 Vastri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package token

type serializedFile struct {
	// Fields correspond 1:1 to fields with same (lower-case) name in File.
	Name  string
	Base  int
	Size  int
	Lines []int
	Infos []lineInfo
}

type serializedFileSet struct {
	Base  int
	Files []serializedFile
}

// Read calls decode to deserialize a file set into s; s must not be nil.
func (s *FileSet) Read(decode func(interface{}) error) error {
	var ss serializedFileSet
	if err := decode(&ss); err != nil {
		return err
	}

	s.mutex.Lock()
	s.base = ss.Base
	files := make([]*File, len(ss.Files))
	for i := 0; i < len(ss.Files); i++ {
		f := &ss.Files[i]
		files[i] = &File{s, f.Name, f.Base, f.Size, f.Lines, f.Infos}
	}
	s.files = files
	s.last = nil
	s.mutex.Unlock()

	return nil
}

// Write calls encode to serialize the file set s.
func (s *FileSet) Write(encode func(interface{}) error) error {
	var ss serializedFileSet

	s.mutex.Lock()
	ss.Base = s.base
	files := make([]serializedFile, len(s.files))
	for i, f := range s.files {
		files[i] = serializedFile{f.name, f.base, f.size, f.lines, f.infos}
	}
	ss.Files = files
	s.mutex.Unlock()

	return encode(ss)
}