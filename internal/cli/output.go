package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"
	"unicode"
)

func (a *App) printJSON(value any) error {
	encoder := json.NewEncoder(a.Stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// printTable renders a result for people. A list becomes one row per item with
// the given columns; anything else becomes a field and value table.
func (a *App) printTable(value any, columns []string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return err
	}

	w := tabwriter.NewWriter(a.Stdout, 0, 4, 2, ' ', 0)
	if rows, ok := listRows(object); ok && len(columns) > 0 {
		headers := make([]string, len(columns))
		for i, column := range columns {
			headers[i] = strings.ToUpper(column)
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
		for _, row := range rows {
			item, _ := row.(map[string]any)
			cells := make([]string, len(columns))
			for i, column := range columns {
				cells[i] = cell(item[column])
			}
			fmt.Fprintln(w, strings.Join(cells, "\t"))
		}
		return w.Flush()
	}

	// Show the fields of a single nested entity at the top level.
	flat := map[string]any{}
	for key, v := range object {
		if nested, ok := v.(map[string]any); ok {
			for k, nv := range nested {
				flat[k] = nv
			}
			continue
		}
		flat[key] = v
	}
	keys := make([]string, 0, len(flat))
	for key := range flat {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	fmt.Fprintln(w, "FIELD\tVALUE")
	for _, key := range keys {
		fmt.Fprintf(w, "%s\t%s\n", cell(key), cell(flat[key]))
	}
	return w.Flush()
}

// listRows returns the items of a list output, which has a single array field.
func listRows(object map[string]any) ([]any, bool) {
	if len(object) != 1 {
		return nil, false
	}
	for _, v := range object {
		rows, ok := v.([]any)
		return rows, ok
	}
	return nil, false
}

// cell formats one value. Names come from the container, so control and
// formatting characters are replaced before they reach the terminal.
func cell(value any) string {
	var text string
	switch v := value.(type) {
	case nil:
	case string:
		text = v
	default:
		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(v); err == nil {
			text = strings.TrimSuffix(b.String(), "\n")
		}
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, text)
}
