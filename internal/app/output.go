package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

type KV struct {
	Key   string
	Value string
}

func PrintJSON(out io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(out, string(data))
	return err
}

func PrintTable(out io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
}

func PrintKV(out io.Writer, title string, rows []KV) {
	if title != "" {
		fmt.Fprintln(out, title)
	}
	for _, row := range rows {
		fmt.Fprintf(out, "  %-10s %s\n", row.Key+":", row.Value)
	}
}

func YesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func HumanDuration(d time.Duration) string {
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(100 * time.Millisecond).String()
}

func HumanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}

	return fmt.Sprintf("%.1f PB", value/unit)
}
