package pdfgen

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/johnfercher/maroto/v2"
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
	gridSize       = 100
	lineNumberGap  = 2.0 // mm between the line number and the code

	pageWidthMM    = 210.0 // A4
	marginMM       = 10.0
	contentWidthMM = pageWidthMM - 2*marginMM
	mmPerGridUnit  = contentWidthMM / gridSize

	// courierGlyphRatio is the fixed advance width of the standard PDF
	// Courier font, expressed as a fraction of its point size (600/1000 em).
	courierGlyphRatio = 0.6
	mmPerPoint        = 25.4 / 72
)

// lineNumberColumns sizes the line-number grid column to exactly fit the
// widest line number in the document (plus lineNumberGap), so the leading
// digit of the longest number lands flush at the left margin, aligned with
// the header text.
func lineNumberColumns(totalLines int) int {
	digits := len(strconv.Itoa(totalLines))
	digitWidthMM := float64(digits) * lineNumberSize * courierGlyphRatio * mmPerPoint
	cols := int(math.Ceil((digitWidthMM + lineNumberGap) / mmPerGridUnit))
	if cols < 1 {
		cols = 1
	}
	return cols
}

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
		WithMaxGridSize(gridSize).
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

	lines := parseCode(body)
	lineNumberCols := lineNumberColumns(len(lines))
	codeCols := gridSize - lineNumberCols

	for i, line := range lines {
		m.AddAutoRow(
			text.NewCol(lineNumberCols, fmt.Sprintf("%d", i+1), props.Text{
				Size:   lineNumberSize,
				Family: fontfamily.Courier,
				Align:  align.Right,
				Right:  lineNumberGap,
			}),
			text.NewCol(codeCols, line, props.Text{
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
