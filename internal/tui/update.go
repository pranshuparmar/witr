package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

type treeMsg model.Result

type debounceMsg struct {
	id  int
	pid int
}

type tickMsg time.Time

func waitTick() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg:
		if m.state == stateList && !m.quitting && !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() {
			cmd = m.refreshProcesses()
			switch m.activeTab {
			case tabPorts:
				cmd = tea.Batch(cmd, m.refreshPorts())
			case tabContainers:
				cmd = tea.Batch(cmd, m.refreshContainers())
			case tabLocks:
				cmd = tea.Batch(cmd, m.refreshLocks())
			}
		}
		return m, tea.Batch(cmd, waitTick())

	case tea.MouseMsg:
		m.statusMsg = "" // clear any transient error on interaction
		if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionMotion && msg.Action != tea.MouseActionRelease {
			return m, nil
		}

		// Required for Windows: isClick is true only for real pointer presses (not scroll wheel).
		isWheel := msg.Button == tea.MouseButtonWheelUp ||
			msg.Button == tea.MouseButtonWheelDown ||
			msg.Button == tea.MouseButtonWheelLeft ||
			msg.Button == tea.MouseButtonWheelRight
		isClick := msg.Action == tea.MouseActionPress && !isWheel

		isDoubleClick := false
		if isClick {
			clickDuration := time.Since(m.lastClickTime)
			if clickDuration < 500*time.Millisecond {
				distX := m.lastClickX - msg.X
				distY := m.lastClickY - msg.Y
				if distX < 0 {
					distX = -distX
				}
				if distY < 0 {
					distY = -distY
				}
				if distX <= 2 && distY <= 1 {
					isDoubleClick = true
				}
			}
			m.lastClickTime = time.Now()
			m.lastClickX = msg.X
			m.lastClickY = msg.Y
		}

		// Handle "witr" Title Click (Home)
		if msg.Y == 1 && isClick && msg.X >= 1 && msg.X <= 6 {
			m.state = stateList
			m.activeTab = tabProcesses
			return m, m.refreshProcesses()
		}

		// Handle Detail View Clicks
		if m.state == stateDetail {
			// Container detail has a single full-width viewport — no env pane,
			// so everything (clicks + wheel) goes straight to m.viewport.
			if m.selectedContainer != nil {
				var cmd tea.Cmd
				detailMsg := msg
				detailMsg.Y -= 3
				detailMsg.X -= 1
				if detailMsg.X >= 0 {
					m.viewport, cmd = m.viewport.Update(detailMsg)
				}
				return m, cmd
			}

			if isClick {
				availableWidth := m.width - 6
				if availableWidth < 0 {
					availableWidth = 0
				}
				detailWidth := int(float64(availableWidth) * 0.7)
				contentX := msg.X - 2

				if contentX < detailWidth {
					m.detailFocus = focusDetail
				} else {
					m.detailFocus = focusEnv
				}
			}

			var cmd tea.Cmd
			detailMsg := msg
			detailMsg.Y -= 3

			if m.detailFocus == focusDetail {
				detailMsg.X -= 1
				if detailMsg.X >= 0 {
					m.viewport, cmd = m.viewport.Update(detailMsg)
				}
			} else {
				availableWidth := m.width - 6
				if availableWidth < 0 {
					availableWidth = 0
				}
				detailWidth := int(float64(availableWidth) * 0.7)
				detailMsg.X -= (detailWidth + 2)
				if detailMsg.X >= 0 {
					m.envViewport, cmd = m.envViewport.Update(detailMsg)
				}
			}
			return m, cmd
		}

		// Handle Search Clicks: blur search if clicking outside the input row
		if isClick && msg.Y != 5 {
			if m.input.Focused() {
				m.input.Blur()
			}
			if m.portInput.Focused() {
				m.portInput.Blur()
			}
			if m.containerInput.Focused() {
				m.containerInput.Blur()
			}
			if m.lockInput.Focused() {
				m.lockInput.Blur()
			}
		}

		// Tabs. X ranges are inactiveTabStyle widths: "1. Processes"=14,
		// "2. Ports"=10, "3. Containers"=15, "4. Locks"=10 (inc. 1ch padding).
		if msg.Y == 1 && isClick {
			if msg.X >= 8 && msg.X < 22 { // "1. Processes"
				if m.activeTab != tabProcesses {
					m.activeTab = tabProcesses
					m.listFocus = focusMain
					return m, nil
				}
			} else if msg.X >= 22 && msg.X < 32 { // "2. Ports"
				if m.activeTab != tabPorts {
					m.activeTab = tabPorts
					m.listFocus = focusMain
					return m, m.refreshPorts()
				}
			} else if msg.X >= 32 && msg.X < 47 { // "3. Containers"
				if m.activeTab != tabContainers {
					m.activeTab = tabContainers
					m.listFocus = focusMain
					return m, m.refreshContainers()
				}
			} else if locksTabEnabled && msg.X >= 47 && msg.X < 57 { // "4. Locks"
				if m.activeTab != tabLocks {
					m.activeTab = tabLocks
					m.listFocus = focusMain
					return m, m.refreshLocks()
				}
			}
		}

		// Handle Search Input Clicks
		if msg.Y == 5 && isClick && m.state == stateList {
			switch m.activeTab {
			case tabProcesses:
				m.input.Focus()
			case tabPorts:
				m.portInput.Focus()
			case tabContainers:
				m.containerInput.Focus()
			case tabLocks:
				m.lockInput.Focus()
			}
			return m, nil
		}

		// Handle Content Area Clicks
		if msg.Y >= 7 {
			contentX := msg.X - 2
			if contentX < 0 {
				return m, nil
			}

			switch m.activeTab {
			case tabProcesses:
				availableWidth := m.width - 6
				processListPaneWidth := int(float64(availableWidth) * 0.7)
				if processListPaneWidth < 10 {
					processListPaneWidth = 10
				}

				if contentX < processListPaneWidth {
					if isClick {
						m.listFocus = focusMain

						if msg.Y == 7 {
							m.handleProcessHeaderClick(contentX)
							return m, nil
						}
					}

					var cmd tea.Cmd
					if isWheel {
						// Convert wheel to key so the table scrolls by one row
						// without jumping the cursor to the mouse Y position.
						var keyMsg tea.KeyMsg
						switch msg.Button {
						case tea.MouseButtonWheelUp:
							keyMsg = tea.KeyMsg{Type: tea.KeyUp}
						case tea.MouseButtonWheelDown:
							keyMsg = tea.KeyMsg{Type: tea.KeyDown}
						}
						prevCursor := m.table.Cursor()
						m.table, cmd = m.table.Update(keyMsg)
						if m.table.Cursor() != prevCursor {
							selected := m.table.SelectedRow()
							if len(selected) > 0 {
								pid := 0
								fmt.Sscanf(selected[0], "%d", &pid)
								m.selectionID++
								id := m.selectionID
								return m, tea.Batch(cmd, tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
									return debounceMsg{id: id, pid: pid}
								}))
							}
						}
						return m, cmd
					}

					tableMsg := msg
					tableMsg.X -= 2
					tableMsg.Y -= 7

					// Manual Row Selection
					if isClick && tableMsg.Y >= 0 {
						view := m.table.View()
						lines := strings.Split(view, "\n")
						if tableMsg.Y < len(lines) {
							line := stripAnsi(lines[tableMsg.Y])
							fields := strings.Fields(line)
							var pid int
							found := false
							for _, f := range fields {
								if p, err := fmt.Sscanf(f, "%d", &pid); err == nil && p > 0 && pid > 0 {
									found = true
									break
								}
							}

							if found {
								targetIdx := -1
								rows := m.table.Rows()
								for i, row := range rows {
									if len(row) > 0 {
										var rowPID int
										if _, err := fmt.Sscanf(row[0], "%d", &rowPID); err == nil && rowPID == pid {
											targetIdx = i
											break
										}
									}
								}
								if targetIdx >= 0 {
									currentIdx := m.table.Cursor()
									diff := targetIdx - currentIdx
									stepKey := tea.KeyMsg{Type: tea.KeyDown}
									if diff < 0 {
										stepKey = tea.KeyMsg{Type: tea.KeyUp}
										diff = -diff
									}
									for j := 0; j < diff; j++ {
										m.table, _ = m.table.Update(stepKey)
									}
								}
							}
						}
					}

					// Row selection check using translated Y
					if isClick && tableMsg.Y > 0 && tableMsg.Y <= m.table.Height() {
						selected := m.table.SelectedRow()
						if len(selected) > 0 {
							pid := 0
							fmt.Sscanf(selected[0], "%d", &pid)

							// Double Click Action: Open Detail
							if isDoubleClick {
								m.state = stateDetail
								m.viewport.GotoTop()
								m.envViewport.GotoTop()
								return m, m.fetchProcessDetail(pid)
							}

							m.selectionID++
							id := m.selectionID
							debounceCmd := tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
								return debounceMsg{id: id, pid: pid}
							})
							return m, tea.Batch(cmd, debounceCmd)
						}
					}
					return m, cmd

				} else {
					if isClick {
						m.listFocus = focusSide
						// Translate click to tree cursor position
						// Offset: border(1) + header(1) + spacer(1) + status(1) + input(1) + table-header-border(1) + table-header(1) + tree-header(1) + tree-header-border(1) + "Ancestry Tree:" label(1) = 10
						treeY := msg.Y - 10
						treeY += m.treeViewport.YOffset
						if treeY >= 0 && treeY < len(m.treePIDs) {
							m.treeCursor = treeY
							m.rerenderTree()

							if isDoubleClick {
								pid := m.treePIDs[m.treeCursor]
								if pid > 0 {
									m.state = stateDetail
									m.viewport.GotoTop()
									m.envViewport.GotoTop()
									return m, m.fetchProcessDetail(pid)
								}
							}
						}
					}
					var cmd tea.Cmd
					// Forward wheel events to viewport for scrolling
					if isWheel {
						treeMsg := msg
						treeMsg.X -= (6 + processListPaneWidth)
						treeMsg.Y -= 8
						if treeMsg.X >= 0 && treeMsg.Y >= 0 {
							m.treeViewport, cmd = m.treeViewport.Update(treeMsg)
						}
					}
					return m, cmd
				}

			case tabPorts:
				availableWidth := m.width - 6
				portPaneWidth := availableWidth / 2

				if contentX < portPaneWidth {
					if isClick {
						m.listFocus = focusMain
						m.portDetailTable.Blur()
						m.portTable.Focus()

						if msg.Y == 7 {
							m.handlePortHeaderClick(contentX)
							return m, nil
						}
					}

					var cmd tea.Cmd
					prevSelected := m.portTable.Cursor()

					if isWheel {
						// Convert wheel to key so the port table scrolls by one row.
						var keyMsg tea.KeyMsg
						switch msg.Button {
						case tea.MouseButtonWheelUp:
							keyMsg = tea.KeyMsg{Type: tea.KeyUp}
						case tea.MouseButtonWheelDown:
							keyMsg = tea.KeyMsg{Type: tea.KeyDown}
						}
						m.portTable, cmd = m.portTable.Update(keyMsg)
						if m.portTable.Cursor() != prevSelected {
							m.updatePortDetails()
						}
						return m, cmd
					}

					// Translate for Port Table
					portMsg := msg
					portMsg.X -= 2
					portMsg.Y -= 7
					if portMsg.X >= 0 && portMsg.Y >= 0 {
						m.portTable, cmd = m.portTable.Update(portMsg)
					}

					// Manual Row Selection for Port Table
					if isClick && portMsg.Y >= 0 {
						view := m.portTable.View()
						lines := strings.Split(view, "\n")
						if portMsg.Y < len(lines) {
							line := stripAnsi(lines[portMsg.Y])
							fields := strings.Fields(line)
							var port int
							var protocol string

							if len(fields) >= 2 {
								if p, err := fmt.Sscanf(fields[0], "%d", &port); err == nil && p > 0 && port > 0 {
									protocol = fields[1]
								}
							}

							if port > 0 {
								rows := m.portTable.Rows()
								for i, row := range rows {
									if len(row) >= 2 {
										if p, err := fmt.Sscanf(row[0], "%d", new(int)); err == nil && p > 0 {
											var rowPort int
											fmt.Sscanf(row[0], "%d", &rowPort)
											if rowPort == port && strings.EqualFold(row[1], protocol) {
												m.portTable.SetCursor(i)
												break
											}
										}
									}
								}
							}
						}
					}

					if m.portTable.Cursor() != prevSelected {
						m.updatePortDetails()
					}

					// Double Click (Ports): Focus Attached Processes
					if isDoubleClick && isClick && portMsg.Y > 0 {
						if portMsg.Y <= m.portTable.Height() {
							m.listFocus = focusSide
							m.portTable.Blur()
							m.portDetailTable.Focus()
							return m, cmd
						}
					}

					return m, cmd

				} else {
					if isClick {
						m.listFocus = focusSide
						m.portTable.Blur()
						m.portDetailTable.Focus()
					}

					var cmd tea.Cmd
					detailMsg := msg
					detailMsg.X -= (4 + portPaneWidth)
					detailMsg.Y -= 9
					if detailMsg.X >= 0 && detailMsg.Y >= 0 {
						m.portDetailTable, cmd = m.portDetailTable.Update(detailMsg)
					}

					if msg.Action == tea.MouseActionPress && detailMsg.Y >= 0 {
						view := m.portDetailTable.View()
						lines := strings.Split(view, "\n")
						if detailMsg.Y < len(lines) {
							line := stripAnsi(lines[detailMsg.Y])
							fields := strings.Fields(line)
							var pid int
							found := false
							for _, f := range fields {
								if p, err := fmt.Sscanf(f, "%d", &pid); err == nil && p > 0 && pid > 0 {
									found = true
									break
								}
							}
							if found {
								rows := m.portDetailTable.Rows()
								for i, row := range rows {
									if len(row) > 0 {
										var rowPID int
										n, _ := fmt.Sscanf(row[0], "%d", &rowPID)
										if n == 1 && rowPID == pid {
											m.portDetailTable.SetCursor(i)
											break
										}
									}
								}
							}
						}
					}

					// Double Click (Attached Processes): Open Detail
					if isDoubleClick && isClick && detailMsg.Y > 0 {
						selected := m.portDetailTable.SelectedRow()
						if len(selected) > 0 {
							pid := 0
							fmt.Sscanf(selected[0], "%d", &pid)
							if pid > 0 {
								m.state = stateDetail
								m.viewport.GotoTop()
								m.envViewport.GotoTop()
								return m, m.fetchProcessDetail(pid)
							}
						}
					}
					return m, cmd
				}

			case tabContainers:
				if isClick {
					m.listFocus = focusMain
					if msg.Y == 7 {
						m.handleContainerHeaderClick(contentX)
						return m, nil
					}
				}

				if isWheel {
					switch msg.Button {
					case tea.MouseButtonWheelUp:
						m.containerTable.MoveUp(1)
					case tea.MouseButtonWheelDown:
						m.containerTable.MoveDown(1)
					}
					return m, nil
				}

				if isClick && msg.Y > 7 {
					// Step 1: move cursor to the clicked row (best-effort row match).
					tableY := msg.Y - 7
					lines := strings.Split(m.containerTable.View(), "\n")
					if tableY < len(lines) {
						clickedSig := normalizeRow(stripAnsi(lines[tableY]))
						for i, row := range m.containerTable.Rows() {
							if normalizeRow(strings.Join(row, " ")) == clickedSig {
								diff := i - m.containerTable.Cursor()
								if diff > 0 {
									m.containerTable.MoveDown(diff)
								} else if diff < 0 {
									m.containerTable.MoveUp(-diff)
								}
								break
							}
						}
					}

					// Step 2 (independent): double-click opens detail for whatever
					// row the cursor is currently on. Even if row matching above
					// failed, this still works as long as the cursor is valid.
					if isDoubleClick {
						idx := m.containerTable.Cursor()
						if idx >= 0 && idx < len(m.filteredContainers) {
							match := m.filteredContainers[idx]
							m.state = stateDetail
							m.selectedDetail = nil
							m.selectedContainer = nil
							m.viewport.GotoTop()
							return m, m.fetchContainerDetail(match)
						}
					}
				}
				return m, nil

			case tabLocks:
				if isClick {
					m.listFocus = focusMain
					if msg.Y == 7 {
						m.handleLockHeaderClick(contentX)
						return m, nil
					}
				}

				if isWheel {
					switch msg.Button {
					case tea.MouseButtonWheelUp:
						m.lockTable.MoveUp(1)
					case tea.MouseButtonWheelDown:
						m.lockTable.MoveDown(1)
					}
					return m, nil
				}

				if isClick && msg.Y > 7 {
					// Step 1: move cursor to the clicked row.
					tableY := msg.Y - 7
					lines := strings.Split(m.lockTable.View(), "\n")
					if tableY < len(lines) {
						clickedSig := normalizeRow(stripAnsi(lines[tableY]))
						for i, row := range m.lockTable.Rows() {
							if normalizeRow(strings.Join(row, " ")) == clickedSig {
								diff := i - m.lockTable.Cursor()
								if diff > 0 {
									m.lockTable.MoveDown(diff)
								} else if diff < 0 {
									m.lockTable.MoveUp(-diff)
								}
								break
							}
						}
					}

					// Step 2 (independent): double-click opens process detail.
					if isDoubleClick {
						idx := m.lockTable.Cursor()
						if idx >= 0 && idx < len(m.filteredLocks) {
							dblPID := m.filteredLocks[idx].PID
							if dblPID > 0 {
								m.state = stateDetail
								m.selectedDetail = nil
								m.selectedContainer = nil
								m.viewport.GotoTop()
								m.envViewport.GotoTop()
								return m, m.fetchProcessDetail(dblPID)
							}
						}
					}
				}
				return m, nil
			}
		}

	case tea.KeyMsg:
		m.statusMsg = "" // clear any transient error on interaction
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "1":
			if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() {
				m.activeTab = tabProcesses
				m.listFocus = focusMain
				return m, nil
			}
		case "2":
			if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() {
				m.activeTab = tabPorts
				m.listFocus = focusMain
				m.portTable.Focus()
				m.portDetailTable.Blur()
				return m, m.refreshPorts()
			}
		case "3":
			if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() {
				m.activeTab = tabContainers
				m.listFocus = focusMain
				return m, m.refreshContainers()
			}
		case "4":
			if locksTabEnabled && !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() {
				m.activeTab = tabLocks
				m.listFocus = focusMain
				return m, m.refreshLocks()
			}
		}

		if m.state == stateList {
			if m.activeTab == tabLocks {
				if m.lockInput.Focused() {
					if msg.String() == "enter" || msg.String() == "esc" {
						m.lockInput.Blur()
						return m, nil
					}
					if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
						m.lockInput.Blur()
					} else {
						var inputCmd tea.Cmd
						m.lockInput, inputCmd = m.lockInput.Update(msg)
						m.updateLockTable()
						m.lockTable.SetCursor(0)
						return m, inputCmd
					}
				}

				if msg.String() == "/" {
					m.lockInput.Focus()
					return m, textinput.Blink
				}
			} else if m.activeTab == tabContainers {
				if m.containerInput.Focused() {
					if msg.String() == "enter" || msg.String() == "esc" {
						m.containerInput.Blur()
						return m, nil
					}
					if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
						m.containerInput.Blur()
					} else {
						var inputCmd tea.Cmd
						m.containerInput, inputCmd = m.containerInput.Update(msg)
						m.updateContainerTable()
						m.containerTable.SetCursor(0)
						return m, inputCmd
					}
				}

				if msg.String() == "/" {
					m.containerInput.Focus()
					return m, textinput.Blink
				}
			} else if m.activeTab == tabPorts {
				if m.portInput.Focused() {
					if msg.String() == "enter" || msg.String() == "esc" {
						m.portInput.Blur()
						return m, nil
					}
					if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
						m.portInput.Blur()
					} else {
						var inputCmd tea.Cmd
						m.portInput, inputCmd = m.portInput.Update(msg)
						m.updatePortTable()
						m.portTable.SetCursor(0)
						return m, inputCmd
					}
				}

				if msg.String() == "/" {
					m.portInput.Focus()
					return m, textinput.Blink
				}
			} else {
				if m.input.Focused() {
					if msg.String() == "enter" || msg.String() == "esc" {
						m.input.Blur()
						return m, nil
					}
					if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
						m.input.Blur()
					} else {
						var inputCmd tea.Cmd
						m.input, inputCmd = m.input.Update(msg)
						m.filterProcesses()

						m.table.SetCursor(0)
						var treeCmd tea.Cmd
						if len(m.filtered) > 0 {
							selected := m.table.SelectedRow()
							if len(selected) > 0 {
								pid := 0
								fmt.Sscanf(selected[0], "%d", &pid)
								m.selectionID++
								id := m.selectionID
								treeCmd = tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
									return debounceMsg{id: id, pid: pid}
								})
							}
						} else {
							m.treeViewport.SetContent("")
						}
						return m, tea.Batch(inputCmd, treeCmd)
					}
				}

				if msg.String() == "/" {
					m.input.Focus()
					return m, textinput.Blink
				}
			}

			switch msg.String() {
			case "q", "Q", "esc":
				m.quitting = true
				return m, tea.Quit
			case "enter":
				if m.activeTab == tabLocks {
					cursor := m.lockTable.Cursor()
					if cursor >= 0 && cursor < len(m.filteredLocks) {
						pid := m.filteredLocks[cursor].PID
						if pid > 0 {
							m.state = stateDetail
							m.selectedDetail = nil
							m.selectedContainer = nil
							m.viewport.GotoTop()
							m.envViewport.GotoTop()
							return m, m.fetchProcessDetail(pid)
						}
					}
				}
				if m.activeTab == tabContainers {
					cursor := m.containerTable.Cursor()
					if cursor >= 0 && cursor < len(m.filteredContainers) {
						match := m.filteredContainers[cursor]
						m.state = stateDetail
						m.selectedDetail = nil
						m.selectedContainer = nil
						m.viewport.GotoTop()
						return m, m.fetchContainerDetail(match)
					}
				}
				if m.activeTab == tabProcesses && m.listFocus == focusMain {
					selected := m.table.SelectedRow()
					if len(selected) > 0 {
						pid := 0
						fmt.Sscanf(selected[0], "%d", &pid)
						if pid > 0 {
							m.state = stateDetail
							m.viewport.GotoTop()
							m.envViewport.GotoTop()
							return m, m.fetchProcessDetail(pid)
						}
					}
				} else if m.activeTab == tabProcesses && m.listFocus == focusSide {
					if m.treeCursor >= 0 && m.treeCursor < len(m.treePIDs) {
						pid := m.treePIDs[m.treeCursor]
						if pid > 0 {
							m.state = stateDetail
							m.viewport.GotoTop()
							m.envViewport.GotoTop()
							return m, m.fetchProcessDetail(pid)
						}
					}
				} else if m.activeTab == tabPorts {
					switch m.listFocus {
					case focusMain:
						m.listFocus = focusSide
						m.portTable.Blur()
						m.portDetailTable.Focus()
					case focusSide:
						selected := m.portDetailTable.SelectedRow()
						if len(selected) > 0 {
							pid := 0
							fmt.Sscanf(selected[0], "%d", &pid)
							if pid > 0 {
								m.state = stateDetail
								m.viewport.GotoTop()
								m.envViewport.GotoTop()
								return m, m.fetchProcessDetail(pid)
							}
						}
					}
				}

			// Focus Switching
			case "tab", "right", "left", "l", "L", "h", "H":
				if m.input.Focused() || m.portInput.Focused() || m.containerInput.Focused() || m.lockInput.Focused() {
					break
				}
				// Tabs without a side panel — these keys are no-ops there.
				if m.activeTab == tabContainers || m.activeTab == tabLocks {
					return m, nil
				}
				if msg.String() == "tab" || msg.String() == "right" || msg.String() == "l" || msg.String() == "L" {
					if m.listFocus == focusMain {
						m.listFocus = focusSide
						if m.activeTab == tabPorts {
							m.portTable.Blur()
							m.portDetailTable.Focus()
						}
					} else {
						m.listFocus = focusMain
						if m.activeTab == tabPorts {
							m.portDetailTable.Blur()
							m.portTable.Focus()
						}
					}
				} else if msg.String() == "shift+tab" || msg.String() == "left" || msg.String() == "h" || msg.String() == "H" {
					if m.listFocus == focusSide {
						m.listFocus = focusMain
						if m.activeTab == tabPorts {
							m.portDetailTable.Blur()
							m.portTable.Focus()
						}
					} else {
						m.listFocus = focusSide
						if m.activeTab == tabPorts {
							m.portTable.Blur()
							m.portDetailTable.Focus()
						}
					}
				}
				return m, nil

			// Toggle All Ports
			case "a", "A":
				if m.activeTab == tabPorts {
					m.showAllPorts = !m.showAllPorts
					m.updatePortTable()
					return m, nil
				}
				if m.activeTab == tabLocks {
					m.showAllFiles = !m.showAllFiles
					m.locks = nil
					m.filteredLocks = nil
					m.lockTable.SetRows(nil)
					m.lockTable.SetCursor(0)
					return m, m.refreshLocks()
				}

			// Sorting Keys (union across all tabs; per-tab dispatch below picks the relevant ones)
			case "c", "C", "p", "P", "n", "N", "m", "M", "t", "T", "u", "U", "s", "S",
				"i", "I", "r", "R", "g", "G", "f", "F":
				switch m.activeTab {
				case tabProcesses:
					newCol := ""
					switch msg.String() {
					case "c", "C":
						newCol = "cpu"
					case "p", "P":
						newCol = "pid"
					case "n", "N":
						newCol = "name"
					case "m", "M":
						newCol = "mem"
					case "t", "T":
						newCol = "time"
					case "u", "U":
						newCol = "user"
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
						cols := m.table.Columns()
						newCols := m.getColumns()
						for i := range cols {
							if i < len(newCols) {
								newCols[i].Width = cols[i].Width
							}
						}
						m.table.SetColumns(newCols)
						return m, nil
					}
				case tabPorts:
					newCol := ""
					switch msg.String() {
					case "p", "P":
						newCol = "port"
					case "t", "T":
						newCol = "proto"
					case "n", "N":
						newCol = "addr"
					case "s", "S":
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
						return m, nil
					}

				case tabContainers:
					newCol := ""
					switch msg.String() {
					case "i", "I":
						newCol = "id"
					case "n", "N":
						newCol = "name"
					case "r", "R":
						newCol = "runtime"
					case "g", "G":
						newCol = "image"
					case "s", "S":
						newCol = "status"
					}
					if newCol != "" {
						if m.sortContainerCol == newCol {
							m.sortContainerDesc = !m.sortContainerDesc
						} else {
							m.sortContainerCol = newCol
							m.sortContainerDesc = false
						}
						m.updateContainerTable()
						return m, nil
					}

				case tabLocks:
					newCol := ""
					switch msg.String() {
					case "p", "P":
						newCol = "pid"
					case "n", "N":
						newCol = "process"
					case "t", "T":
						newCol = "type"
					case "m", "M":
						newCol = "mode"
					case "f", "F":
						newCol = "path"
					}
					if newCol != "" {
						if m.sortLockCol == newCol {
							m.sortLockDesc = !m.sortLockDesc
						} else {
							m.sortLockCol = newCol
							m.sortLockDesc = false
						}
						m.updateLockTable()
						return m, nil
					}
				}
			}

			// Table navigation or Tree scrolling
			var cmd tea.Cmd
			if m.listFocus == focusMain {
				if m.activeTab == tabProcesses {
					prevSelected := -1
					if len(m.filtered) > 0 {
						prevSelected = m.table.Cursor()
					}

					m.table, cmd = m.table.Update(msg)

					if len(m.filtered) > 0 && m.table.Cursor() != prevSelected {
						selected := m.table.SelectedRow()
						if len(selected) > 0 {
							idx := m.table.Cursor()
							if idx >= 0 && idx < len(m.filtered) {
								m.selectionID++
								id := m.selectionID
								p := m.filtered[idx]
								debounceCmd := tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
									return debounceMsg{id: id, pid: p.PID}
								})
								return m, tea.Batch(cmd, debounceCmd)
							}
						}
					}
					return m, cmd
				} else if m.activeTab == tabContainers {
					m.containerTable, cmd = m.containerTable.Update(msg)
					return m, cmd
				} else if m.activeTab == tabLocks {
					m.lockTable, cmd = m.lockTable.Update(msg)
					return m, cmd
				} else {
					prevSelected := m.portTable.Cursor()
					m.portTable, cmd = m.portTable.Update(msg)
					if m.portTable.Cursor() != prevSelected {
						m.updatePortDetails()
					}
					return m, cmd
				}
			} else {
				if m.activeTab == tabProcesses {
					switch msg.Type {
					case tea.KeyUp:
						if m.treeCursor > 0 {
							m.treeCursor--
							m.rerenderTree()
						}
					case tea.KeyDown:
						if m.treeCursor < len(m.treePIDs)-1 {
							m.treeCursor++
							m.rerenderTree()
						}
					case tea.KeyEnter:
						if m.treeCursor >= 0 && m.treeCursor < len(m.treePIDs) {
							pid := m.treePIDs[m.treeCursor]
							if pid > 0 {
								m.state = stateDetail
								m.viewport.GotoTop()
								m.envViewport.GotoTop()
								return m, m.fetchProcessDetail(pid)
							}
						}
					default:
						m.treeViewport, cmd = m.treeViewport.Update(msg)
					}
				} else {
					m.portDetailTable, cmd = m.portDetailTable.Update(msg)
				}
				return m, cmd
			}

		} else if m.state == stateDetail {
			pid := 0
			if m.selectedDetail != nil {
				pid = m.selectedDetail.Process.PID
			}

			// renice text input
			if m.pendingAction == actionRenice {
				switch msg.String() {
				case "esc":
					m.pendingAction = actionNone
					m.reniceInput.SetValue("")
					m.reniceInput.Blur()
				case "enter":
					val := 0
					if _, err := fmt.Sscanf(m.reniceInput.Value(), "%d", &val); err != nil {
						m.statusMsg = "Invalid nice value — enter a number between −20 and 19"
					} else if err := setNice(pid, val); err != nil {
						m.statusMsg = fmt.Sprintf("Renice failed: %v", err)
					} else {
						m.statusMsg = fmt.Sprintf("PID %d reniced to %d", pid, val)
					}
					m.pendingAction = actionNone
					m.reniceInput.SetValue("")
					m.reniceInput.Blur()
				default:
					var inputCmd tea.Cmd
					m.reniceInput, inputCmd = m.reniceInput.Update(msg)
					return m, inputCmd
				}
				return m, nil
			}

			// confirmation prompt
			if m.pendingAction != actionNone {
				switch msg.String() {
				case "y", "Y":
					originalAction := m.pendingAction
					var execErr error
					switch originalAction {
					case actionKill:
						execErr = killProcess(pid)
					case actionTerm:
						execErr = termProcess(pid)
					case actionPause:
						execErr = pauseProcess(pid)
					case actionResume:
						execErr = resumeProcess(pid)
					}
					m.pendingAction = actionNone
					if execErr != nil {
						m.statusMsg = fmt.Sprintf("Error: %v", execErr)
						return m, nil
					}
					switch originalAction {
					case actionKill, actionTerm:
						// Process is gone — go back to list
						m.state = stateList
						m.selectedDetail = nil
						m.statusMsg = fmt.Sprintf("Signal sent to PID %d", pid)
						return m, m.refreshProcesses()
					default:
						// Pause/Resume succeeded — stay in detail view
						m.statusMsg = "Done"
						return m, nil
					}
				case "n", "N", "esc":
					m.pendingAction = actionNone
				}
				return m, nil
			}

			// action menu
			if m.actionMenuOpen {
				switch msg.String() {
				case "k", "K":
					m.actionMenuOpen = false
					m.pendingAction = actionKill
				case "t", "T":
					m.actionMenuOpen = false
					m.pendingAction = actionTerm
				case "p", "P":
					m.actionMenuOpen = false
					m.pendingAction = actionPause
				case "r", "R":
					m.actionMenuOpen = false
					m.pendingAction = actionResume
				case "n", "N":
					m.actionMenuOpen = false
					m.pendingAction = actionRenice
					m.reniceInput.Focus()
					return m, textinput.Blink
				case "esc", "q", "Q":
					m.actionMenuOpen = false
				}
				return m, nil
			}

			// detail view navigation
			switch msg.String() {
			case "esc", "q", "Q", "backspace":
				m.state = stateList
				m.selectedDetail = nil
				m.selectedContainer = nil
				m.detailFocus = focusDetail
				m.actionMenuOpen = false
				m.pendingAction = actionNone
				m.reniceInput.SetValue("")
				m.reniceInput.Blur()
				if m.activeTab == tabContainers {
					return m, m.refreshContainers()
				}
				return m, m.refreshProcesses()
			case "a", "A":
				if actionsSupported && m.selectedDetail != nil {
					m.actionMenuOpen = true
				}
				return m, nil
			case "left", "h", "H":
				m.detailFocus = focusDetail
				return m, nil
			case "right", "l", "L":
				m.detailFocus = focusEnv
				return m, nil
			case "tab":
				if m.detailFocus == focusDetail {
					m.detailFocus = focusEnv
				} else {
					m.detailFocus = focusDetail
				}
				return m, nil
			default:
				var cmd tea.Cmd
				if m.detailFocus == focusDetail {
					m.viewport, cmd = m.viewport.Update(msg)
				} else {
					m.envViewport, cmd = m.envViewport.Update(msg)
				}
				return m, cmd
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		availableWidth := msg.Width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}

		processListHeight := msg.Height - 11
		if processListHeight < 5 {
			processListHeight = 5
		}

		processListPaneWidth := int(float64(availableWidth) * 0.7)
		if processListPaneWidth < 10 {
			processListPaneWidth = 10
		}

		processTablePadding := 4
		processListWidth := processListPaneWidth - processTablePadding
		if processListWidth < 10 {
			processListWidth = 10
		}

		fixedColumnsWidth := 81 // PID(8)+Name(20)+User(12)+CPU(6)+Mem(16)+Started(19)
		cmdWidth := processListWidth - fixedColumnsWidth - 12
		m.showCmdCol = cmdWidth >= 15
		if cmdWidth < 10 {
			cmdWidth = 10
		}

		columns := m.getColumns()
		if m.showCmdCol {
			columns[6].Width = cmdWidth
		}
		m.table.SetWidth(processListWidth)
		m.table.SetHeight(processListHeight)
		m.table.SetRows(nil)
		m.table.SetColumns(columns)
		m.filterProcesses()

		treeWidth := availableWidth - processListPaneWidth - 4
		if treeWidth < 10 {
			treeWidth = 10
		}
		m.treeViewport.Width = treeWidth
		m.treeViewport.Height = processListHeight - 2
		if m.treeViewport.Height < 0 {
			m.treeViewport.Height = 0
		}
		portListHeight := processListHeight

		portPaneWidth := int(float64(availableWidth) * 0.5)
		if portPaneWidth < 0 {
			portPaneWidth = 0
		}

		tablePadding := 4
		portTableWidth := portPaneWidth - tablePadding
		if portTableWidth < 0 {
			portTableWidth = 0
		}

		portColumns := m.getPortColumns()
		fixedPortWidth := 36
		buffer := 6
		addrWidth := portTableWidth - fixedPortWidth - buffer
		if addrWidth < 10 {
			addrWidth = 10
		}
		if len(portColumns) > 2 {
			portColumns[2].Width = addrWidth
		}
		m.portTable.SetColumns(portColumns)
		m.portTable.SetWidth(portTableWidth)
		m.portTable.SetHeight(portListHeight)

		portDetailWidth := availableWidth - portPaneWidth - 5
		if portDetailWidth < 10 {
			portDetailWidth = 10
		}

		pdCols := m.portDetailTable.Columns()
		fixedPdWidth := 35
		buffer = 6
		cmdPdWidth := portDetailWidth - fixedPdWidth - buffer
		if cmdPdWidth < 10 {
			cmdPdWidth = 10
		}
		if len(pdCols) > 3 {
			pdCols[3].Width = cmdPdWidth
		}
		// Center-align PID header in port detail table
		if len(pdCols) > 0 {
			pdCols[0].Title = centerHeader("PID", pdCols[0].Width)
		}
		m.portDetailTable.SetColumns(pdCols)
		m.portDetailTable.SetWidth(portDetailWidth)
		m.portDetailTable.SetHeight(portListHeight - 2)

		m.containerTable.SetWidth(availableWidth)
		m.containerTable.SetHeight(processListHeight)
		// Size the trailing Command column so total column widths match the
		// table width — bubbles table draws the header underline across
		// SetWidth, so column overflow leaves a partial border.
		containerCols := m.containerTable.Columns()
		fixedCC := 0
		const cmdColIdx = 6
		for i, c := range containerCols {
			if i == cmdColIdx {
				continue
			}
			fixedCC += c.Width
		}
		const cellPadding = 2 // tableHeaderStyle Padding(0, 1) → 1 char per side
		paddingBudget := cellPadding * len(containerCols)
		if cmdColIdx < len(containerCols) {
			cmdWidth := availableWidth - fixedCC - paddingBudget
			if cmdWidth < 10 {
				cmdWidth = 10
			}
			containerCols[cmdColIdx].Width = cmdWidth
			m.containerTable.SetColumns(containerCols)
		}

		m.lockTable.SetWidth(availableWidth)
		m.lockTable.SetHeight(processListHeight)
		// Last column (Path) absorbs remaining width, same pattern as Command.
		lockCols := m.lockTable.Columns()
		fixedLC := 0
		pathColIdx := len(lockCols) - 1
		for i, c := range lockCols {
			if i == pathColIdx {
				continue
			}
			fixedLC += c.Width
		}
		if pathColIdx >= 0 {
			pathWidth := availableWidth - fixedLC - cellPadding*len(lockCols)
			if pathWidth < 10 {
				pathWidth = 10
			}
			lockCols[pathColIdx].Width = pathWidth
			m.lockTable.SetColumns(lockCols)
		}

		vpHeight := msg.Height - 9
		if vpHeight < 0 {
			vpHeight = 0
		}
		detailViewWidth := int(float64(availableWidth) * 0.7)
		envViewWidth := availableWidth - detailViewWidth - 4

		m.viewport.Width = detailViewWidth - 4
		m.viewport.Height = vpHeight

		m.envViewport.Width = envViewWidth
		if m.envViewport.Width < 0 {
			m.envViewport.Width = 0
		}
		m.envViewport.Height = vpHeight

		m.updatePortDetails()

	case []model.Process:
		var currentPID int
		selectedRow := m.table.SelectedRow()
		if len(selectedRow) > 0 {
			fmt.Sscanf(selectedRow[0], "%d", &currentPID)
		}

		m.processes = msg
		m.sortProcesses()
		m.filterProcesses()

		newIdx := 0
		found := false
		if currentPID > 0 {
			for i, p := range m.filtered {
				if p.PID == currentPID {
					newIdx = i
					found = true
					break
				}
			}
		}

		if len(m.filtered) > 0 {
			if !found {
				newIdx = 0
			}
			m.table.SetCursor(newIdx)

			m.selectionID++
			p := m.filtered[newIdx]
			return m, m.fetchTree(p)
		}

	case debounceMsg:
		if msg.id == m.selectionID {
			var targetProc model.Process
			found := false
			row := m.table.SelectedRow()
			if len(row) > 0 {
				var pID int
				fmt.Sscanf(row[0], "%d", &pID)
				if pID == msg.pid {
					idx := m.table.Cursor()
					if idx >= 0 && idx < len(m.filtered) {
						targetProc = m.filtered[idx]
						found = true
					}
				}
			}
			if !found {
				for _, p := range m.processes {
					if p.PID == msg.pid {
						targetProc = p
						found = true
						break
					}
				}
			}

			if found {
				return m, m.fetchTree(targetProc)
			}
		}

	case []model.OpenPort:
		m.ports = msg
		m.updatePortTable()
		m.updatePortDetails()

	case []*model.ContainerMatch:
		m.containers = msg
		m.updateContainerTable()

	case []*model.LockedFile:
		m.locks = msg
		m.updateLockTable()

	case treeMsg:
		selected := m.table.SelectedRow()
		if len(selected) > 0 {
			var currentPID int
			fmt.Sscanf(selected[0], "%d", &currentPID)
			if model.Result(msg).Process.PID == currentPID {
				m.updateTreeViewport(model.Result(msg))
			}
		}

	case model.Result:
		m.selectedDetail = &msg
		m.selectedContainer = nil
		m.updateDetailViewport()
		m.updateEnvViewport()

	case *model.ContainerMatch:
		m.selectedContainer = msg
		m.selectedDetail = nil
		m.updateDetailViewport()

	case error:
		// Revert to list view on any error
		m.state = stateList
		m.selectedDetail = nil
		m.selectedContainer = nil
		m.statusMsg = fmt.Sprintf("Error: %v", msg)
		return m, m.refreshProcesses()
	}

	return m, nil
}
