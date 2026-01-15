package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// TableInfo represents a database table for display.
type TableInfo struct {
	Name        string
	Namespace   string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	ForeignKeys []ForeignKeyInfo
	Docs        string
}

// QualifiedName returns namespace.name or just name if no namespace.
func (t TableInfo) QualifiedName() string {
	if t.Namespace != "" {
		return t.Namespace + "." + t.Name
	}
	return t.Name
}

// ColumnInfo represents a column for display.
type ColumnInfo struct {
	Name     string
	Type     string
	TypeArgs string
	Nullable bool
	Unique   bool
	IsPK     bool
	Default  string
	Docs     string
}

// IndexInfo represents an index for display.
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKeyInfo represents a foreign key for display.
type ForeignKeyInfo struct {
	Name      string
	Column    string
	RefTable  string
	RefColumn string
	OnDelete  string
	OnUpdate  string
}

// DriftItem represents a schema drift for display.
type DriftItem struct {
	Type    string // missing, extra, modified
	Object  string // table or column name
	Details string
}

// MigrationItem represents a migration for the status view.
type MigrationItem struct {
	Revision   string
	Name       string
	Status     string // applied, pending, missing, modified
	AppliedAt  string
	ExecTimeMs int
	Checksum   string
	FilePath   string
	SQL        string // for preview
}

// StatusData holds all data for the unified status TUI.
type StatusData struct {
	Revision  string
	GitBranch string

	// Tab 1: Browse
	Tables []TableInfo

	// Tab 2: History
	Migrations []MigrationItem

	// Tab 3: Verify
	GitStatus  []string // uncommitted files
	LockHolder string   // who holds the migration lock, if any

	// Tab 4: Drift
	DriftItems []DriftItem
}

// ShowStatus displays the unified status TUI with 4 tabs.
// initialTab can be "browse", "history", "verify", or "drift".
func ShowStatus(initialTab string, data StatusData) {
	if !isTerminal() {
		showStatusSimple(initialTab, data)
		return
	}

	app := tview.NewApplication()
	pages := tview.NewPages()

	// Track current tab
	currentTab := initialTab
	if currentTab == "" {
		currentTab = TabBrowse
	}

	// Panel focus tracking for Browse tab (H/L navigation)
	var browsePanels []tview.Primitive
	var browsePanelIndex int

	// Create all 4 tab views
	browseView, panels := createStatusBrowseTab(app, data)
	browsePanels = panels
	historyView := createStatusHistoryTab(app, data)
	verifyView := createStatusVerifyTab(app, data)
	driftView := createStatusDriftTab(app, data)

	pages.AddPage(TabBrowse, browseView, true, currentTab == TabBrowse)
	pages.AddPage(TabHistory, historyView, true, currentTab == TabHistory)
	pages.AddPage(TabVerify, verifyView, true, currentTab == TabVerify)
	pages.AddPage(TabDrift, driftView, true, currentTab == TabDrift)

	// Tab bar
	tabLabels := []string{TabLabelBrowse, TabLabelHistory, TabLabelVerify, TabLabelDrift}
	tabBar := createStatusTabBar(currentTab, tabLabels)

	// Header (centralized config)
	applied, pending := countMigrationStats(data.Migrations)
	header := tview.NewTextView().
		SetText(FormatHeader(data.Revision, applied, pending)).
		SetTextColor(Theme.Text).
		SetTextAlign(tview.AlignLeft)
	header.SetBackgroundColor(Theme.Primary)

	// Status bar
	statusBar := tview.NewTextView().
		SetText(HintsStatus).
		SetTextColor(Theme.TextDim).
		SetTextAlign(tview.AlignCenter)
	statusBar.SetBackgroundColor(Theme.Background)

	// Spacer between header and tabs
	spacer := tview.NewBox().SetBackgroundColor(Theme.Background)

	// Main layout
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, Layout.HeaderHeight, 0, false).
		AddItem(spacer, Layout.SpacerHeight, 0, false).
		AddItem(tabBar, Layout.TabBarHeight, 0, false).
		AddItem(pages, 0, 1, true).
		AddItem(statusBar, Layout.StatusBarHeight, 0, false)
	layout.SetBackgroundColor(Theme.Background)

	// Helper to update tab bar
	updateTabBar := func(tab string) {
		currentTab = tab
		tabBar.Clear()
		writeStatusTabBarContent(tabBar, tabLabels, tab)
	}

	// Global keyboard shortcuts
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		case '1':
			pages.SwitchToPage(TabBrowse)
			updateTabBar(TabBrowse)
			if len(browsePanels) > 0 {
				app.SetFocus(browsePanels[browsePanelIndex])
			}
			return nil
		case '2':
			pages.SwitchToPage(TabHistory)
			updateTabBar(TabHistory)
			return nil
		case '3':
			pages.SwitchToPage(TabVerify)
			updateTabBar(TabVerify)
			return nil
		case '4':
			pages.SwitchToPage(TabDrift)
			updateTabBar(TabDrift)
			return nil
		// Panel navigation (H/L for Browse tab)
		case 'h':
			if currentTab == TabBrowse && len(browsePanels) > 0 {
				browsePanelIndex--
				if browsePanelIndex < 0 {
					browsePanelIndex = len(browsePanels) - 1
				}
				app.SetFocus(browsePanels[browsePanelIndex])
				return nil
			}
		case 'l':
			if currentTab == TabBrowse && len(browsePanels) > 0 {
				browsePanelIndex++
				if browsePanelIndex >= len(browsePanels) {
					browsePanelIndex = 0
				}
				app.SetFocus(browsePanels[browsePanelIndex])
				return nil
			}
		// Vim-style navigation
		case 'j':
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case 'k':
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case 'g':
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		case 'G':
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		}
		// Arrow key panel navigation
		if currentTab == TabBrowse && len(browsePanels) > 0 {
			switch event.Key() {
			case tcell.KeyLeft:
				browsePanelIndex--
				if browsePanelIndex < 0 {
					browsePanelIndex = len(browsePanels) - 1
				}
				app.SetFocus(browsePanels[browsePanelIndex])
				return nil
			case tcell.KeyRight:
				browsePanelIndex++
				if browsePanelIndex >= len(browsePanels) {
					browsePanelIndex = 0
				}
				app.SetFocus(browsePanels[browsePanelIndex])
				return nil
			}
		}
		if event.Key() == tcell.KeyEscape {
			app.Stop()
			return nil
		}
		return event
	})

	app.SetRoot(layout, true).EnableMouse(true).Run()
}

// createStatusTabBar creates the tab bar for the unified status view.
func createStatusTabBar(activeTab string, tabLabels []string) *tview.TextView {
	bar := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	bar.SetBackgroundColor(Theme.Background)
	writeStatusTabBarContent(bar, tabLabels, activeTab)
	return bar
}

// writeStatusTabBarContent writes tab bar content with the active tab highlighted.
func writeStatusTabBarContent(bar *tview.TextView, tabs []string, activeTab string) {
	// Map tab names to indices
	tabMap := map[string]int{TabBrowse: 0, TabHistory: 1, TabVerify: 2, TabDrift: 3}
	activeIndex := tabMap[activeTab]

	var text strings.Builder
	text.WriteString(" ")
	for i, tab := range tabs {
		if i == activeIndex {
			text.WriteString(fmt.Sprintf(TagSelected+" %s "+TagEnd+" ", tab))
		} else {
			text.WriteString(fmt.Sprintf(" %s  ", tab))
		}
	}
	bar.SetText(text.String())
}

// createStatusBrowseTab creates the Browse tab with 3-panel layout.
func createStatusBrowseTab(app *tview.Application, data StatusData) (tview.Primitive, []tview.Primitive) {
	if len(data.Tables) == 0 {
		// Empty state
		empty := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("\n\n" + TagMuted + MsgNoTables + "\n\n" + MsgRunNewMigration + TagReset)
		empty.SetBackgroundColor(Theme.Background)
		return empty, nil
	}

	// Left panel: table list
	tableList := tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetSelectedBackgroundColor(Theme.Selection).
		SetSelectedTextColor(Theme.Highlight).
		SetSelectedFocusOnly(false).
		SetMainTextColor(Theme.Text)
	tableList.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelTables).
		SetTitleColor(Theme.Accent)

	// Center panel: columns table
	columnsTable := tview.NewTable().
		SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(Theme.Highlight).
			Background(Theme.Selection))
	columnsTable.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelColumns).
		SetTitleColor(Theme.Accent)

	// Right panel: details (indexes, FKs)
	details := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	details.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelDetails).
		SetTitleColor(Theme.Accent)

	// Populate table list
	for _, t := range data.Tables {
		tableList.AddItem(t.QualifiedName(), "", 0, nil)
	}

	// Update columns and details when table selected
	tableList.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		if index < len(data.Tables) {
			populateColumnsTable(columnsTable, data.Tables[index])
			populateTableDetails(details, data.Tables[index])
		}
	})

	// Initialize with first table
	tableList.SetCurrentItem(0)
	populateColumnsTable(columnsTable, data.Tables[0])
	populateTableDetails(details, data.Tables[0])

	// 3-column grid: Tables | Columns (widest) | Details
	grid := tview.NewFlex().
		AddItem(tableList, Layout.BrowseTablesWidth, 0, true).
		AddItem(columnsTable, 0, 1, false).
		AddItem(details, Layout.BrowseDetailsWidth, 0, false)
	grid.SetBackgroundColor(Theme.Background)

	// Return panels for focus switching
	panels := []tview.Primitive{tableList, columnsTable, details}
	return grid, panels
}

// createStatusHistoryTab creates the History tab with applied migrations.
func createStatusHistoryTab(app *tview.Application, data StatusData) tview.Primitive {
	// Filter to only applied migrations
	var applied []MigrationItem
	for _, m := range data.Migrations {
		if m.Status == StatusApplied {
			applied = append(applied, m)
		}
	}

	if len(applied) == 0 {
		empty := tview.NewTextView().
			SetDynamicColors(true).
			SetTextAlign(tview.AlignCenter).
			SetText("\n\n" + TagMuted + MsgNoMigrationsYet + "\n\n" + MsgRunMigrate + TagReset)
		empty.SetBackgroundColor(Theme.Background)
		return empty
	}

	// Left: history list table
	list := tview.NewTable().
		SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(Theme.Highlight).
			Background(Theme.Selection))
	list.SetBackgroundColor(Theme.Background)

	// Headers with expansion
	expansions := []int{Layout.TableExpandOther, Layout.TableExpandMain, Layout.TableExpandMain, Layout.TableExpandOther}
	for i, h := range ColsHistory {
		list.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(Theme.Header).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold).
			SetExpansion(expansions[i]))
	}

	// Right: details panel
	details := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	details.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelDetails).
		SetTitleColor(Theme.Accent)

	// Calculate total exec time
	var totalExecMs int

	// Populate list (newest first - reverse order)
	for i := len(applied) - 1; i >= 0; i-- {
		m := applied[i]
		row := len(applied) - i
		totalExecMs += m.ExecTimeMs

		list.SetCell(row, 0, tview.NewTableCell(m.Revision).
			SetTextColor(Theme.Accent).
			SetAttributes(tcell.AttrBold).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandOther))
		list.SetCell(row, 1, tview.NewTableCell(m.Name).
			SetTextColor(Theme.Text).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandMain))
		list.SetCell(row, 2, tview.NewTableCell(m.AppliedAt).
			SetTextColor(Theme.TextDim).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandMain))

		// Exec time with color coding (centered)
		execTime := fmt.Sprintf("%dms", m.ExecTimeMs)
		execColor := Theme.Success
		if m.ExecTimeMs >= Layout.ExecTimeWarningMs {
			execColor = Theme.Warning
			execTime = fmt.Sprintf("%.2fs", float64(m.ExecTimeMs)/1000)
		}
		list.SetCell(row, 3, tview.NewTableCell(execTime).
			SetTextColor(execColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandOther))
	}

	// Select first row and show details
	list.Select(1, 0)
	details.SetText(formatMigrationDetails(applied[len(applied)-1]))

	// Update details on selection change
	list.SetSelectionChangedFunc(func(row, col int) {
		idx := len(applied) - row
		if row > 0 && idx >= 0 && idx < len(applied) {
			m := applied[idx]
			details.SetText(formatMigrationDetails(m))
		}
	})

	// Summary text at bottom
	var totalTime string
	if totalExecMs >= Layout.ExecTimeWarningMs {
		totalTime = fmt.Sprintf("%.2fs", float64(totalExecMs)/1000)
	} else {
		totalTime = fmt.Sprintf("%dms", totalExecMs)
	}
	summary := tview.NewTextView().
		SetText(fmt.Sprintf(" Total: %d migrations (%s)", len(applied), totalTime)).
		SetTextColor(Theme.TextDim)
	summary.SetBackgroundColor(Theme.Background)

	// Grid layout with summary
	contentFlex := tview.NewFlex().
		AddItem(list, 0, Layout.HistoryListRatio, true).
		AddItem(details, 0, Layout.HistoryDetailsRatio, false)
	contentFlex.SetBackgroundColor(Theme.Background)

	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(contentFlex, 0, 1, true).
		AddItem(summary, Layout.SummaryHeight, 0, false)
	layout.SetBackgroundColor(Theme.Background)

	return layout
}

// createStatusVerifyTab creates the Verify tab with chain integrity and git status.
func createStatusVerifyTab(app *tview.Application, data StatusData) tview.Primitive {
	// Left panel: Chain integrity
	chainPanel := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	chainPanel.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelChainIntegrity).
		SetTitleColor(Theme.Accent)

	// Build chain integrity content
	var chainContent strings.Builder
	chainContent.WriteString(TagLabel + TitleMigrationChainStatus + TagValue + "\n\n")

	applied, pending := countMigrationStats(data.Migrations)
	issues := 0
	for _, m := range data.Migrations {
		if m.Status == StatusMissing || m.Status == StatusModified {
			issues++
		}
	}

	chainContent.WriteString(fmt.Sprintf(TagSuccess+LabelApplied+TagValue+" %d\n", applied))
	chainContent.WriteString(fmt.Sprintf(TagLabel+LabelPending+TagValue+" %d\n", pending))
	if issues > 0 {
		chainContent.WriteString(fmt.Sprintf(TagError+LabelIssues+TagValue+" %d\n", issues))
	}

	chainContent.WriteString("\n" + TagLabel + LabelChecksums + TagValue + "\n")
	for _, m := range data.Migrations {
		statusIcon := TagSuccess + SymbolCheck + TagReset
		if m.Status == StatusPending {
			statusIcon = TagLabel + SymbolCircle + TagReset
		} else if m.Status == StatusMissing || m.Status == StatusModified {
			statusIcon = TagError + SymbolCross + TagReset
		}
		checksum := m.Checksum
		if len(checksum) > Layout.ChecksumDisplayLen {
			checksum = checksum[:Layout.ChecksumDisplayLen]
		}
		chainContent.WriteString(fmt.Sprintf("%s %s "+TagMuted+"%s"+TagReset+"\n", statusIcon, m.Revision, checksum))
	}

	chainPanel.SetText(chainContent.String())

	// Right panel: Git status
	gitPanel := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	gitPanel.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelGitStatus).
		SetTitleColor(Theme.Accent)

	// Build git status content
	var gitContent strings.Builder
	if data.GitBranch != "" {
		gitContent.WriteString(fmt.Sprintf(TagLabel+LabelBranch+TagValue+" %s\n\n", data.GitBranch))

		if len(data.GitStatus) == 0 {
			gitContent.WriteString(TagSuccess + SymbolCheck + " " + MsgAllCommitted + TagValue + "\n")
		} else {
			gitContent.WriteString(TagLabel + LabelUncommitted + TagValue + "\n")
			for _, file := range data.GitStatus {
				gitContent.WriteString(fmt.Sprintf("  "+TagError+"•"+TagReset+" %s\n", file))
			}
			gitContent.WriteString("\n" + TagMuted + MsgRunGitAdd + TagReset + "\n")
		}
	} else {
		gitContent.WriteString(TagLabel + MsgNotGitRepo + TagValue + "\n\n")
		gitContent.WriteString(TagMuted + MsgGitVersionControl + TagReset + "\n")
	}

	if data.LockHolder != "" {
		gitContent.WriteString(fmt.Sprintf("\n"+TagLabel+LabelLock+TagValue+" "+FmtLockHeldBy+"\n", data.LockHolder))
	}

	gitPanel.SetText(gitContent.String())

	// Grid layout: chain + git
	grid := tview.NewFlex().
		AddItem(chainPanel, 0, Layout.VerifyChainRatio, true).
		AddItem(gitPanel, 0, Layout.VerifyGitRatio, false)
	grid.SetBackgroundColor(Theme.Background)

	return grid
}

// createStatusDriftTab creates the Drift tab with drift items and details.
func createStatusDriftTab(app *tview.Application, data StatusData) tview.Primitive {
	// Left: drift items list
	list := tview.NewTable().
		SetBorders(true).
		SetFixed(1, 0).
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(Theme.Highlight).
			Background(Theme.Selection))
	list.SetBackgroundColor(Theme.Background)

	// Headers with expansion
	driftExpansions := []int{Layout.TableExpandOther, Layout.TableExpandMain, Layout.TableExpandMain}
	for i, h := range ColsDrift {
		list.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(Theme.Header).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold).
			SetExpansion(driftExpansions[i]))
	}

	// Right: details panel
	details := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)
	details.SetBackgroundColor(Theme.Background).
		SetBorder(true).
		SetBorderColor(Theme.Border).
		SetTitle(PanelDriftDetails).
		SetTitleColor(Theme.Accent)

	// Populate drift items
	if len(data.DriftItems) == 0 {
		// No drift - show success message
		details.SetText(TagSuccess + DriftMsgNoDrift + TagValue + "\n\n" + DriftMsgSchemaMatches)
	} else {
		for i, d := range data.DriftItems {
			row := i + 1
			// Type with color
			typeColor := Theme.Warning
			typeIcon := DriftIconModified
			if d.Type == DriftMissing {
				typeColor = Theme.Error
				typeIcon = DriftIconMissing
			} else if d.Type == DriftExtra {
				typeColor = Theme.Warning
				typeIcon = DriftIconExtra
			}
			list.SetCell(row, 0, tview.NewTableCell(typeIcon+" "+d.Type).
				SetTextColor(typeColor).
				SetAlign(tview.AlignCenter).
				SetExpansion(Layout.TableExpandOther))
			list.SetCell(row, 1, tview.NewTableCell(d.Object).
				SetTextColor(Theme.Accent).
				SetExpansion(Layout.TableExpandMain))
			list.SetCell(row, 2, tview.NewTableCell(d.Details).
				SetTextColor(Theme.TextDim).
				SetExpansion(Layout.TableExpandMain))
		}

		// Select first row
		list.Select(1, 0)

		// Update details on selection
		list.SetSelectionChangedFunc(func(row, col int) {
			if row > 0 && row <= len(data.DriftItems) {
				d := data.DriftItems[row-1]
				details.SetText(formatDriftDetails(d))
			}
		})

		// Initialize details
		details.SetText(formatDriftDetails(data.DriftItems[0]))
	}

	// Grid layout
	grid := tview.NewFlex().
		AddItem(list, 0, Layout.DriftListRatio, true).
		AddItem(details, 0, Layout.DriftDetailsRatio, false)
	grid.SetBackgroundColor(Theme.Background)

	return grid
}

// showStatusSimple displays simple non-interactive output.
func showStatusSimple(tab string, data StatusData) {
	switch tab {
	case TabBrowse:
		fmt.Println(RenderTitle(fmt.Sprintf(FmtSchemaAtRev, data.Revision)))
		fmt.Println()
		if len(data.Tables) == 0 {
			fmt.Println(Info(MsgNoTables))
		} else {
			fmt.Printf("  %s\n\n", Muted(FormatCount(len(data.Tables), "table", "tables")))
			for _, t := range data.Tables {
				fmt.Printf("  %s\n", Primary(t.QualifiedName()))
				for _, col := range t.Columns {
					fmt.Printf("    - %s %s\n", col.Name, Dim(col.Type))
				}
				if len(t.ForeignKeys) > 0 {
					fmt.Printf("    %s\n", Muted(LabelForeignKeys))
					for _, fk := range t.ForeignKeys {
						fmt.Printf("      %s -> %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn)
					}
				}
				fmt.Println()
			}
		}

	case TabHistory:
		fmt.Println(RenderTitle(TitleMigrationHistory))
		fmt.Println()
		for i := len(data.Migrations) - 1; i >= 0; i-- {
			m := data.Migrations[i]
			if m.Status == StatusApplied {
				fmt.Printf("  %s  %-30s  %s  (%dms)\n",
					m.Revision, m.Name, Dim(m.AppliedAt), m.ExecTimeMs)
			}
		}

	case TabVerify:
		fmt.Println(RenderTitle(TitleMigrationVerify))
		fmt.Println()
		applied, pending := countMigrationStats(data.Migrations)
		fmt.Printf("  Applied: %d\n", applied)
		fmt.Printf("  Pending: %d\n", pending)
		if data.GitBranch != "" {
			fmt.Printf("  Branch:  %s\n", data.GitBranch)
		}
		// Checksums
		fmt.Println()
		for _, m := range data.Migrations {
			icon := SymbolCheck
			if m.Status == StatusPending {
				icon = SymbolCircle
			} else if m.Status == StatusMissing || m.Status == StatusModified {
				icon = SymbolCross
			}
			checksum := m.Checksum
			if len(checksum) > Layout.ChecksumDisplayLen {
				checksum = checksum[:Layout.ChecksumDisplayLen]
			}
			fmt.Printf("  %s %s %s\n", icon, m.Revision, Dim(checksum))
		}

	case TabDrift:
		fmt.Println(RenderTitle(TitleSchemaDrift))
		fmt.Println()
		if len(data.DriftItems) == 0 {
			fmt.Println(Success(DriftMsgNoDrift))
		} else {
			for _, d := range data.DriftItems {
				status := d.Type
				switch d.Type {
				case DriftMissing:
					status = Error(DriftIconMissing + " " + status)
				case DriftExtra:
					status = Warning(DriftIconExtra + " " + status)
				default:
					status = Info(DriftIconModified + " " + status)
				}
				fmt.Printf("  %s  %s\n", status, d.Object)
			}
		}

	default:
		// Default: show migration status
		fmt.Println(RenderTitle(TitleMigrationStatus))
		fmt.Println()
		for _, m := range data.Migrations {
			status := m.Status
			switch m.Status {
			case StatusApplied:
				status = Success(status)
			case StatusPending:
				status = Warning(status)
			default:
				status = Error(status)
			}
			fmt.Printf("  %s  %-30s  %s\n", m.Revision, m.Name, status)
		}
	}
	fmt.Println()
}

// ShowStatusText displays text-only output for all tabs.
func ShowStatusText(data StatusData) {
	// Schema at revision
	fmt.Println(RenderTitle(fmt.Sprintf(FmtSchemaAtRev, data.Revision)))
	fmt.Println()
	if len(data.Tables) == 0 {
		fmt.Println(Info(MsgNoTables))
	} else {
		fmt.Printf("  %s\n\n", Muted(FormatCount(len(data.Tables), "table", "tables")))
		for _, t := range data.Tables {
			fmt.Printf("  %s\n", Primary(t.QualifiedName()))
			for _, col := range t.Columns {
				fmt.Printf("    - %s %s\n", col.Name, Dim(col.Type))
			}
			if len(t.ForeignKeys) > 0 {
				fmt.Printf("    %s\n", Muted(LabelForeignKeys))
				for _, fk := range t.ForeignKeys {
					fmt.Printf("      %s -> %s.%s\n", fk.Column, fk.RefTable, fk.RefColumn)
				}
			}
			fmt.Println()
		}
	}

	// Migration History
	fmt.Println(RenderTitle(TitleMigrationHistory))
	fmt.Println()
	hasApplied := false
	for i := len(data.Migrations) - 1; i >= 0; i-- {
		m := data.Migrations[i]
		if m.Status == StatusApplied {
			hasApplied = true
			fmt.Printf("  %s  %-30s  %s  (%dms)\n",
				m.Revision, m.Name, Dim(m.AppliedAt), m.ExecTimeMs)
		}
	}
	if !hasApplied {
		fmt.Println("  " + MsgNone)
	}
	fmt.Println()

	// Verification
	fmt.Println(RenderTitle(TitleVerification))
	fmt.Println()
	applied, pending := countMigrationStats(data.Migrations)
	fmt.Printf("  Applied: %d\n", applied)
	fmt.Printf("  Pending: %d\n", pending)
	if data.GitBranch != "" {
		fmt.Printf("  Branch:  %s\n", data.GitBranch)
	}
	// Check for issues
	issues := 0
	for _, m := range data.Migrations {
		if m.Status == StatusMissing || m.Status == StatusModified {
			issues++
		}
	}
	if issues == 0 {
		fmt.Printf("  %s %s\n", SymbolCheck, MsgAllChecksumsValid)
	} else {
		fmt.Printf("  %s "+FmtIssuesFound+"\n", SymbolCross, issues)
	}
	fmt.Println()

	// Schema Drift
	fmt.Println(RenderTitle(TitleSchemaDrift))
	fmt.Println()
	if len(data.DriftItems) == 0 {
		fmt.Println("  " + MsgNoDriftDetected)
	} else {
		for _, d := range data.DriftItems {
			icon := DriftIconModified
			if d.Type == DriftMissing {
				icon = DriftIconMissing
			} else if d.Type == DriftExtra {
				icon = DriftIconExtra
			}
			fmt.Printf("  %s %s  %s\n", icon, d.Type, d.Object)
		}
	}
	fmt.Println()
}

// countMigrationStats counts applied and pending migrations.
func countMigrationStats(migrations []MigrationItem) (applied, pending int) {
	for _, m := range migrations {
		if m.Status == StatusApplied {
			applied++
		} else {
			pending++
		}
	}
	return
}

// populateColumnsTable populates the columns table for a given table.
func populateColumnsTable(table *tview.Table, t TableInfo) {
	table.Clear()

	// Headers with expansion
	for i, h := range ColsColumns {
		cell := tview.NewTableCell(h).
			SetTextColor(Theme.Header).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		// Name and Type columns expand more
		if i <= 1 {
			cell.SetExpansion(Layout.TableExpandMain)
		} else {
			cell.SetExpansion(Layout.TableExpandOther)
		}
		table.SetCell(0, i, cell)
	}

	// Data
	for i, col := range t.Columns {
		row := i + 1

		// Name (expanded)
		table.SetCell(row, 0, tview.NewTableCell(col.Name).
			SetTextColor(Theme.Accent).
			SetAttributes(tcell.AttrBold).
			SetExpansion(Layout.TableExpandMain))

		// Type with args (centered)
		colType := col.Type
		if col.TypeArgs != "" {
			colType = fmt.Sprintf("%s(%s)", col.Type, col.TypeArgs)
		}
		table.SetCell(row, 1, tview.NewTableCell(colType).
			SetTextColor(Theme.Info).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandMain))

		// Nullable (centered)
		nullable := KeyYes
		nullColor := Theme.TextDim
		if !col.Nullable {
			nullable = KeyNo
			nullColor = Theme.Error
		}
		table.SetCell(row, 2, tview.NewTableCell(nullable).
			SetTextColor(nullColor).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandOther))

		// Key (centered)
		key := ""
		if col.IsPK {
			key = KeyPK
		} else if col.Unique {
			key = KeyUQ
		}
		table.SetCell(row, 3, tview.NewTableCell(key).
			SetTextColor(Theme.Warning).
			SetAlign(tview.AlignCenter).
			SetExpansion(Layout.TableExpandOther))
	}
}

// populateTableDetails populates the details panel for a table.
func populateTableDetails(view *tview.TextView, t TableInfo) {
	var text strings.Builder

	if t.Docs != "" {
		text.WriteString(fmt.Sprintf(TagMuted+"%s"+TagReset+"\n\n", t.Docs))
	}

	// Indexes
	text.WriteString(TagLabel + LabelIndexes + TagValue + "\n")
	if len(t.Indexes) == 0 {
		text.WriteString("  " + TagMuted + MsgNone + TagReset + "\n")
	} else {
		for _, idx := range t.Indexes {
			uniqueStr := ""
			if idx.Unique {
				uniqueStr = " " + TagLabel + KeyUnique + TagReset
			}
			text.WriteString(fmt.Sprintf("  - %s [%s]%s\n",
				idx.Name,
				strings.Join(idx.Columns, ", "),
				uniqueStr))
		}
	}

	// Foreign Keys
	text.WriteString("\n" + TagLabel + LabelForeignKeys + TagValue + "\n")
	if len(t.ForeignKeys) == 0 {
		text.WriteString("  " + TagMuted + MsgNone + TagReset + "\n")
	} else {
		for _, fk := range t.ForeignKeys {
			text.WriteString(fmt.Sprintf("  - %s -> %s.%s\n",
				fk.Column, fk.RefTable, fk.RefColumn))
			if fk.OnDelete != "" || fk.OnUpdate != "" {
				text.WriteString(fmt.Sprintf("    "+TagMuted+LabelOnDelete+" %s, "+LabelOnUpdate+" %s"+TagReset+"\n",
					fk.OnDelete, fk.OnUpdate))
			}
		}
	}

	view.SetText(text.String())
}

// formatMigrationDetails formats migration details for the detail panel.
func formatMigrationDetails(m MigrationItem) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf(TagLabel+LabelRevision+TagValue+" %s\n", m.Revision))
	details.WriteString(fmt.Sprintf(TagLabel+LabelName+TagValue+" %s\n", m.Name))

	// Status with color
	statusTag := TagLabel
	switch m.Status {
	case StatusApplied:
		statusTag = TagSuccess
	case StatusMissing, StatusModified:
		statusTag = TagError
	}
	details.WriteString(fmt.Sprintf(TagLabel+LabelStatus+statusTag+" %s"+TagValue+"\n", m.Status))

	if m.AppliedAt != "" {
		details.WriteString(fmt.Sprintf(TagLabel+LabelApplied+TagValue+" %s\n", m.AppliedAt))
	}
	if m.ExecTimeMs > 0 {
		details.WriteString(fmt.Sprintf(TagLabel+LabelDuration+TagValue+" %dms\n", m.ExecTimeMs))
	}
	if m.Checksum != "" {
		details.WriteString(fmt.Sprintf(TagLabel+LabelChecksum+TagValue+" %s\n", m.Checksum))
	}
	if m.FilePath != "" {
		details.WriteString(fmt.Sprintf(TagLabel+LabelFile+TagValue+" %s\n", m.FilePath))
	}

	if m.SQL != "" {
		details.WriteString("\n" + TagLabel + LabelSQLPreview + TagValue + "\n")
		// Truncate SQL if too long
		sql := m.SQL
		if len(sql) > Layout.SQLPreviewMaxLen {
			sql = sql[:Layout.SQLPreviewMaxLen] + "..."
		}
		details.WriteString(sql)
	}

	return details.String()
}

// formatDriftDetails formats drift item details for the details panel.
func formatDriftDetails(d DriftItem) string {
	var text strings.Builder

	typeTag := TagLabel
	typeIcon := DriftIconModified
	switch d.Type {
	case DriftMissing:
		typeTag = TagError
		typeIcon = DriftIconMissing
	case DriftExtra:
		typeTag = TagLabel
		typeIcon = DriftIconExtra
	}

	text.WriteString(fmt.Sprintf(TagLabel+LabelType+typeTag+" %s %s"+TagValue+"\n", typeIcon, d.Type))
	text.WriteString(fmt.Sprintf(TagLabel+LabelObject+TagValue+" %s\n", d.Object))
	text.WriteString(fmt.Sprintf("\n"+TagLabel+LabelDetails+TagValue+"\n%s\n", d.Details))

	// Help text
	text.WriteString("\n" + TagMuted)
	switch d.Type {
	case DriftMissing:
		text.WriteString(DriftHelpMissing)
	case DriftExtra:
		text.WriteString(DriftHelpExtra)
	case DriftModified:
		text.WriteString(DriftHelpModified)
	}
	text.WriteString(TagReset)

	return text.String()
}
