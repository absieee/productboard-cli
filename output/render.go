package output

import (
	"encoding/json"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// Table writes a formatted ASCII table to w.
func Table(w io.Writer, headers []string, rows [][]string) {
	t := tablewriter.NewTable(w,
		tablewriter.WithHeader(headers),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)
	for _, row := range rows {
		_ = t.Append(row)
	}
	_ = t.Render()
}

// JSON marshals v as indented JSON to w.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Stderr writes an error message to stderr.
func Stderr(msg string) {
	_, _ = os.Stderr.WriteString("error: " + msg + "\n")
}
