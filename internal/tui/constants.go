package tui

import "time"

// Timing constants for the interactive TUI.
const (
	// refreshInterval is the auto-refresh cadence for the process/port/
	// container/lock/network tabs. 3s mirrors top's default.
	refreshInterval = 3 * time.Second

	// selectionDebounce delays the detail/tree fetch after the selection
	// moves, so holding a cursor key down doesn't spawn a fetch per row.
	selectionDebounce = 500 * time.Millisecond

	// doubleClickThreshold is the maximum gap between two clicks for them to
	// count as a double-click.
	doubleClickThreshold = 500 * time.Millisecond
)

// Pane split ratios — the fraction of the available width given to the primary
// pane. These are shared by the resize math, the mouse hit-testing, and the
// view renderer; keeping them in one place stops those three from drifting
// apart (which would make clicks land on the wrong pane).
const (
	listPaneRatio    = 0.7 // process list vs. ancestry tree
	detailPaneRatio  = 0.7 // detail view vs. environment view
	portPaneRatio    = 0.5 // port list vs. port detail
	networkPaneRatio = 0.5 // network list vs. attached containers
)

// Adaptive auto-refresh. The background-refresh cadence starts at
// refreshInterval and adapts to the measured refresh cost: after backoffStreak
// consecutive refreshes over slowFraction of the interval it grows by
// refreshStep (up to maxRefreshInterval); after backoffStreak under
// fastFraction it shrinks by refreshStep (back down to refreshInterval). The
// band between is stable, so it can't oscillate. All internal — no config.
const (
	maxRefreshInterval = 30 * time.Second // ceiling
	refreshStep        = 3 * time.Second  // grow/shrink increment
	backoffStreak      = 2                // consecutive samples before adjusting
	slowFraction       = 0.6              // refresh over this fraction of interval => slow
	fastFraction       = 0.3              // refresh under this fraction => fast
)
