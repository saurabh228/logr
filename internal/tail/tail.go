package tail

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"time"

	"github.com/saurabh/logr/internal/filter"
	"github.com/saurabh/logr/internal/parser"
	"github.com/saurabh/logr/internal/render"
)

// Follow reads path from the beginning, then keeps watching for new lines
// as they are appended — behaves like tail -f. Handles log rotation (truncation).
func Follow(path string, engine *filter.Engine, opts render.Options, parseOpts parser.Options, out io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := bytes.TrimRight(line, "\r\n")
			if len(trimmed) > 0 {
				entry := parser.ParseWith(trimmed, parseOpts)
				if engine.Pass(entry) {
					render.Render(entry, out, opts)
				}
			}
		}
		if err == io.EOF {
			// Check for log rotation: if file shrank, reopen from the start.
			info, statErr := os.Stat(path)
			if statErr == nil {
				cur, _ := f.Seek(0, io.SeekCurrent)
				if info.Size() < cur {
					f.Seek(0, io.SeekStart)
					reader.Reset(f)
				}
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}
