package output

import (
	"io"
	"sort"

	"github.com/confluentinc/go-printer"
)

type HumanListWriter struct {
	outputFormat output
	data         [][]string
	listFields   []string
	listLabels   []string
	writer       io.Writer
}

func (o *HumanListWriter) AddMapElement(e map[string]string) {
	var data []string
	for _, field := range o.listFields {
		data = append(data, e[field])
	}
	o.data = append(o.data, data)
}

func (o *HumanListWriter) AddElement(e interface{}) {
	o.data = append(o.data, printer.ToRow(e, o.listFields))
}

func (o *HumanListWriter) Out() error {
	printer.RenderCollectionTableOut(o.data, o.listLabels, o.writer)
	return nil
}

func (o *HumanListWriter) GetOutputFormat() output {
	return o.outputFormat
}

func (o *HumanListWriter) StableSort() {
	sort.Slice(o.data, func(i, j int) bool {
		for x := range o.data[i] {
			if o.data[i][x] != o.data[j][x] {
				return o.data[i][x] < o.data[j][x]
			}
		}
		return false
	})
}
