// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goldmark

import (
	"bytes"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/gohugoio/hugo/common/text"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"

	bp "github.com/gohugoio/hugo/bufferpool"
)

func sanitizeAnchorNameString(s string, asciiOnly bool) string {
	return string(sanitizeAnchorName([]byte(s), asciiOnly))
}

func sanitizeAnchorName(b []byte, asciiOnly bool) []byte {
	return sanitizeAnchorNameWithHook(b, asciiOnly, nil)
}

func sanitizeAnchorNameWithHook(b []byte, asciiOnly bool, hook func(buf *bytes.Buffer)) []byte {
	buf := bp.GetBuffer()

	if asciiOnly {
		// Normalize it to preserve accents if possible.
		b = text.RemoveAccents(b)
	}

	for len(b) > 0 {
		r, size := utf8.DecodeRune(b)
		switch {
		case asciiOnly && size != 1:
		case r == '-' || isSpace(r):
			buf.WriteRune('-')
		case isAlphaNumeric(r):
			buf.WriteRune(unicode.ToLower(r))
		default:
		}

		b = b[size:]
	}

	if hook != nil {
		hook(buf)
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())

	bp.PutBuffer(buf)

	return result
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

var _ parser.IDs = (*idFactory)(nil)

type idFactory struct {
	asciiOnly bool
	vals      map[string]struct{}
}

func newIDFactory(asciiOnly bool) *idFactory {
	return &idFactory{
		vals:      make(map[string]struct{}),
		asciiOnly: asciiOnly,
	}
}

func (ids *idFactory) Generate(value []byte, kind ast.NodeKind) []byte {
	return sanitizeAnchorNameWithHook(value, ids.asciiOnly, func(buf *bytes.Buffer) {
		if buf.Len() == 0 {
			if kind == ast.KindHeading {
				buf.WriteString("heading")
			} else {
				buf.WriteString("id")
			}
		}

		if _, found := ids.vals[util.BytesToReadOnlyString(buf.Bytes())]; found {
			// Append a hypen and a number, starting with 1.
			buf.WriteRune('-')
			pos := buf.Len()
			for i := 1; ; i++ {
				buf.WriteString(strconv.Itoa(i))
				if _, found := ids.vals[util.BytesToReadOnlyString(buf.Bytes())]; !found {
					break
				}
				buf.Truncate(pos)
			}
		}

		ids.vals[buf.String()] = struct{}{}

	})
}

func (ids *idFactory) Put(value []byte) {
	ids.vals[util.BytesToReadOnlyString(value)] = struct{}{}
}