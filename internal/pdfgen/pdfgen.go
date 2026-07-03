package pdfgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

const (
	fontSize       = 9.0
	lineNumberSize = 8.0
)

// Header groups the metadata printed at the top of every page.
type Header struct {
	Username string
	TeamName string
	TeamID   string
	Location string
}

// Generate renders body as a monospaced, line-numbered PDF with a repeating
// header, mirroring the behavior of the legacy jsPDF implementation.
func Generate(body string, header Header) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithTopMargin(10).
		WithRightMargin(10).
		WithBottomMargin(10).
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} / {total}",
			Place:   props.RightTop,
			Family:  fontfamily.Courier,
			Size:    fontSize,
		}).
		WithDefaultFont(&props.Font{
			Family: fontfamily.Courier,
			Size:   fontSize,
		}).
		Build()

	m := maroto.New(cfg)

	headerLine := fmt.Sprintf(
		"user: %s | team: %s (%s) | location: %s",
		header.Username, header.TeamName, header.TeamID, header.Location,
	)
	if err := m.RegisterHeader(text.NewRow(10, headerLine, props.Text{
		Size:   fontSize,
		Family: fontfamily.Courier,
		Align:  align.Left,
	})); err != nil {
		return nil, fmt.Errorf("register pdf header: %w", err)
	}

	for i, line := range parseCode(body) {
		m.AddAutoRow(
			text.NewCol(1, fmt.Sprintf("%d", i+1), props.Text{
				Size:   lineNumberSize,
				Family: fontfamily.Courier,
				Align:  align.Right,
			}),
			col.New(1),
			text.NewCol(10, line, props.Text{
				Size:   fontSize,
				Family: fontfamily.Courier,
				Align:  align.Left,
			}),
		)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return doc.GetBytes(), nil
}

// PageCount reports how many pages a generated PDF contains.
func PageCount(pdf []byte) (int, error) {
	count, err := api.PageCount(bytes.NewReader(pdf), nil)
	if err != nil {
		return 0, fmt.Errorf("count pdf pages: %w", err)
	}
	return count, nil
}

// parseCode mirrors the legacy TypeScript ParseCode: tabs become four spaces,
// a single trailing newline is trimmed, and lines are split on newlines that
// are not escaped with a preceding backslash.
func parseCode(input string) []string {
	expanded := strings.ReplaceAll(input, "\t", "    ")
	expanded = strings.TrimSuffix(expanded, "\n")

	runes := []rune(expanded)
	var lines []string
	var cur strings.Builder
	for i, r := range runes {
		if r == '\n' && (i == 0 || runes[i-1] != '\\') {
			lines = append(lines, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	lines = append(lines, cur.String())

	return lines
}
