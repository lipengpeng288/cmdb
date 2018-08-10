// Copyright © 2018 Alfred Chou <unioverlord@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"bytes"
	"io"

	tablewriter "github.com/olekukonko/tablewriter"
)

// ASCIITable is a shorthand of tablewriter
type ASCIITable struct {
	align   int
	headers []string
	rows    [][]string
	writer  io.Writer
}

// Save writes generated ASCII table into inner writer
func (in *ASCIITable) Save() {
	table := tablewriter.NewWriter(in.writer)
	table.SetHeader(in.headers)
	table.SetAlignment(in.align)
	table.AppendBulk(in.rows)
	table.Render()
}

// NewASCIITable returns a fulfilled ASCIITable
func NewASCIITable(headers []string, align int, writer io.Writer, rows ...[]string) *ASCIITable {
	return &ASCIITable{
		align:   align,
		headers: headers,
		rows:    rows,
		writer:  writer,
	}
}

// SimpleASCIITable is a shorthand of NewASCIITable, but returns a rendered bytes array.
func SimpleASCIITable(headers []string, fields ...[]string) []byte {
	var buf bytes.Buffer
	NewASCIITable(headers, tablewriter.ALIGN_LEFT, &buf, fields...).Save()
	return buf.Bytes()
}
