package markdown

import (
	"strings"

	ext "github.com/yuin/goldmark/extension/ast"
)

func (r *renderer) renderTableLines(n *ext.Table, source []byte, availableWidth int, nextKind string) []renderLine {
	textLines := r.renderTable(n, source, availableWidth, nextKind)
	out := make([]renderLine, len(textLines))
	for i, line := range textLines {
		out[i] = rl(line, true)
	}
	return out
}

func (r *renderer) renderTable(n *ext.Table, source []byte, availableWidth int, nextKind string) []string {
	headerCells := r.tableHeaderCells(n, source)
	bodyRows := r.tableBodyRows(n, source)
	numCols := len(headerCells)
	if numCols == 0 {
		return nil
	}

	borderOverhead := 3*numCols + 1
	availableForCells := availableWidth - borderOverhead
	if availableForCells < numCols {
		raw := r.tableRawFallback(n, source)
		lines := wrapTextWithANSI(raw, availableWidth)
		if nextKind != "" && nextKind != "space" {
			lines = append(lines, "")
		}
		return lines
	}

	maxUnbrokenWordWidth := 30
	naturalWidths := make([]int, numCols)
	minWordWidths := make([]int, numCols)

	for i, cell := range headerCells {
		naturalWidths[i] = visibleWidth(cell)
		minWordWidths[i] = max(1, longestWordWidth(cell, maxUnbrokenWordWidth))
	}
	for _, row := range bodyRows {
		for i, cell := range row {
			if i >= numCols {
				break
			}
			naturalWidths[i] = max(naturalWidths[i], visibleWidth(cell))
			minWordWidths[i] = max(minWordWidths[i], longestWordWidth(cell, maxUnbrokenWordWidth))
		}
	}

	minColumnWidths := append([]int(nil), minWordWidths...)
	minCellsWidth := sum(minColumnWidths)
	if minCellsWidth > availableForCells {
		minColumnWidths = make([]int, numCols)
		for i := range minColumnWidths {
			minColumnWidths[i] = 1
		}
		remaining := availableForCells - numCols
		if remaining > 0 {
			totalWeight := 0
			for _, w := range minWordWidths {
				totalWeight += max(0, w-1)
			}
			growth := make([]int, numCols)
			for i, w := range minWordWidths {
				weight := max(0, w-1)
				if totalWeight > 0 {
					growth[i] = (weight * remaining) / totalWeight
				}
			}
			for i := range minColumnWidths {
				minColumnWidths[i] += growth[i]
			}
			allocated := sum(growth)
			leftover := remaining - allocated
			for i := 0; leftover > 0 && i < numCols; i++ {
				minColumnWidths[i]++
				leftover--
			}
		}
		minCellsWidth = sum(minColumnWidths)
	}

	totalNaturalWidth := sum(naturalWidths) + borderOverhead
	columnWidths := make([]int, numCols)
	if totalNaturalWidth <= availableWidth {
		for i := range columnWidths {
			columnWidths[i] = max(naturalWidths[i], minColumnWidths[i])
		}
	} else {
		totalGrowPotential := 0
		for i, w := range naturalWidths {
			totalGrowPotential += max(0, w-minColumnWidths[i])
		}
		extraWidth := max(0, availableForCells-minCellsWidth)
		for i := range columnWidths {
			minWidth := minColumnWidths[i]
			naturalWidth := naturalWidths[i]
			grow := 0
			if totalGrowPotential > 0 {
				grow = (max(0, naturalWidth-minWidth) * extraWidth) / totalGrowPotential
			}
			columnWidths[i] = minWidth + grow
		}
		remaining := availableForCells - sum(columnWidths)
		for remaining > 0 {
			grew := false
			for i := range columnWidths {
				if remaining <= 0 {
					break
				}
				if columnWidths[i] < naturalWidths[i] {
					columnWidths[i]++
					remaining--
					grew = true
				}
			}
			if !grew {
				break
			}
		}
	}

	var lines []string
	topBorder := boxBorder("┌", "┬", "┐", columnWidths)
	lines = append(lines, topBorder)

	headerLines := wrapCells(headerCells, columnWidths)
	headerLineCount := maxLines(headerLines)
	for lineIdx := 0; lineIdx < headerLineCount; lineIdx++ {
		parts := make([]string, numCols)
		for col := 0; col < numCols; col++ {
			text := cellLine(headerLines, col, lineIdx)
			parts[col] = padCell(r.theme.Bold(text), columnWidths[col])
		}
		lines = append(lines, "│ "+strings.Join(parts, " │ ")+" │")
	}

	separator := boxBorder("├", "┼", "┤", columnWidths)
	lines = append(lines, separator)

	for rowIdx, row := range bodyRows {
		rowLines := wrapCells(row, columnWidths)
		rowLineCount := maxLines(rowLines)
		for lineIdx := 0; lineIdx < rowLineCount; lineIdx++ {
			parts := make([]string, numCols)
			for col := 0; col < numCols; col++ {
				text := cellLine(rowLines, col, lineIdx)
				parts[col] = padCell(text, columnWidths[col])
			}
			lines = append(lines, "│ "+strings.Join(parts, " │ ")+" │")
		}
		if rowIdx < len(bodyRows)-1 {
			lines = append(lines, separator)
		}
	}

	lines = append(lines, boxBorder("└", "┴", "┘", columnWidths))
	if nextKind != "" && nextKind != "space" {
		lines = append(lines, "")
	}
	return lines
}

func (r *renderer) tableHeaderCells(n *ext.Table, source []byte) []string {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		header, ok := c.(*ext.TableHeader)
		if !ok {
			continue
		}
		var cells []string
		for cell := header.FirstChild(); cell != nil; cell = cell.NextSibling() {
			tc, ok := cell.(*ext.TableCell)
			if !ok {
				continue
			}
			cells = append(cells, r.renderBlockText(tc, source))
		}
		return cells
	}
	return nil
}

func (r *renderer) tableBodyRows(n *ext.Table, source []byte) [][]string {
	var rows [][]string
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		tr, ok := c.(*ext.TableRow)
		if !ok {
			continue
		}
		rows = append(rows, r.tableRowCells(tr, source))
	}
	return rows
}

func (r *renderer) tableRowCells(row *ext.TableRow, source []byte) []string {
	var cells []string
	for c := row.FirstChild(); c != nil; c = c.NextSibling() {
		cell, ok := c.(*ext.TableCell)
		if !ok {
			continue
		}
		cells = append(cells, r.renderBlockText(cell, source))
	}
	return cells
}

func (r *renderer) tableRawFallback(n *ext.Table, source []byte) string {
	var b strings.Builder
	header := r.tableHeaderCells(n, source)
	if len(header) > 0 {
		b.WriteString("| ")
		b.WriteString(strings.Join(header, " | "))
		b.WriteString(" |\n")
	}
	for _, row := range r.tableBodyRows(n, source) {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func wrapCells(cells []string, columnWidths []int) [][]string {
	out := make([][]string, len(cells))
	for i, cell := range cells {
		width := 1
		if i < len(columnWidths) {
			width = columnWidths[i]
		}
		out[i] = wrapTextWithANSI(cell, max(1, width))
	}
	return out
}

func cellLine(cellLines [][]string, col, lineIdx int) string {
	if col >= len(cellLines) || lineIdx >= len(cellLines[col]) {
		return ""
	}
	return cellLines[col][lineIdx]
}

func padCell(text string, width int) string {
	pad := width - visibleWidth(text)
	if pad <= 0 {
		return text
	}
	return text + strings.Repeat(" ", pad)
}

func boxBorder(left, mid, right string, widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("─", w)
	}
	return left + "─" + strings.Join(parts, mid+"─") + "─" + right
}

func longestWordWidth(text string, maxWidth int) int {
	longest := 0
	for _, word := range strings.Fields(text) {
		w := visibleWidth(word)
		if w > longest {
			longest = w
		}
	}
	if maxWidth > 0 && longest > maxWidth {
		return maxWidth
	}
	return longest
}

func maxLines(cellLines [][]string) int {
	max := 0
	for _, lines := range cellLines {
		if len(lines) > max {
			max = len(lines)
		}
	}
	return max
}

func sum(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
