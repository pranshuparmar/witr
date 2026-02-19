package tui

import "github.com/charmbracelet/bubbles/table"

// returns the column index at x pixels, or -1 if not found.
func (m *MainModel) getColumnAtX(x int, cols []table.Column) int {
	currentX := 0
	for i, col := range cols {
		colWidth := col.Width + 2
		if x >= currentX && x < currentX+colWidth {
			return i
		}
		currentX += colWidth
	}
	return -1
}

func (m *MainModel) handleProcessHeaderClick(x int) {
	cols := m.table.Columns()
	colIdx := m.getColumnAtX(x, cols)

	if colIdx >= 0 {
		newCol := ""
		switch colIdx {
		case 0:
			newCol = "pid"
		case 1:
			newCol = "user"
		case 2:
			newCol = "name"
		case 3:
			newCol = "cpu"
		case 4:
			newCol = "mem"
		case 5:
			newCol = "time"
		}

		if newCol != "" {
			if m.sortCol == newCol {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortCol = newCol
				m.sortDesc = true
			}
			m.sortProcesses()
			m.filterProcesses()

			newCols := m.getColumns()
			for i := range cols {
				if i < len(newCols) {
					newCols[i].Width = cols[i].Width
				}
			}
			m.table.SetColumns(newCols)
		}
	}
}

func (m *MainModel) handlePortHeaderClick(x int) {
	cols := m.portTable.Columns()
	colIdx := m.getColumnAtX(x, cols)

	if colIdx >= 0 {
		newCol := ""
		switch colIdx {
		case 0:
			newCol = "port"
		case 1:
			newCol = "proto"
		case 2:
			newCol = "addr"
		case 3:
			newCol = "state"
		}

		if newCol != "" {
			if m.sortPortCol == newCol {
				m.sortPortDesc = !m.sortPortDesc
			} else {
				m.sortPortCol = newCol
				m.sortPortDesc = false
			}
			m.updatePortTable()
		}
	}
}
