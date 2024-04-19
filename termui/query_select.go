package termui

import (
	"fmt"
	"sort"

	"github.com/awesome-gocui/gocui"
)

const querySelectView = "querySelectView"

var querySelectHelp = helpBar{
	{"q", "Close"},
	{"↓↑,jk", "Nav"},
	{"↵", "Select query"},
}

type querySelect struct {
	searchKeys sort.StringSlice
	searches   map[string]string
	selected   int
	scroll     int
	childViews []string
}

func newQuerySelect() *querySelect {
	return &querySelect{}
}

func (qs *querySelect) SetSearches(searches map[string]string) {
	qs.searches = searches
	qs.searchKeys = nil
	for key := range searches {
		qs.searchKeys = append(qs.searchKeys, key)
	}
	sort.Sort(qs.searchKeys)
	qs.selected = 0
	qs.scroll = 0
}

func (qs *querySelect) keybindings(g *gocui.Gui) error {
	// Abort
	if err := g.SetKeybinding(querySelectView, gocui.KeyEsc, gocui.ModNone, qs.abort); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'q', gocui.ModNone, qs.abort); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(querySelectView, gocui.KeyArrowUp, gocui.ModNone, qs.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'k', gocui.ModNone, qs.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(querySelectView, gocui.KeyArrowDown, gocui.ModNone, qs.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'j', gocui.ModNone, qs.selectNext); err != nil {
		return err
	}
	// Select
	if err := g.SetKeybinding(querySelectView, gocui.KeySpace, gocui.ModNone, qs.selectAndReturn); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'x', gocui.ModNone, qs.selectAndReturn); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, gocui.KeyEnter, gocui.ModNone, qs.selectAndReturn); err != nil {
		return err
	}
	return nil
}

func (qs *querySelect) layout(g *gocui.Gui) error {
	// Clean up child views to support scrolling
	for _, view := range qs.childViews {
		if err := g.DeleteView(view); err != nil && !gocui.IsUnknownView(err) {
			return err
		}
	}
	qs.childViews = nil

	maxX, maxY := g.Size()
	width := minInt(30, maxX)
	lines := len(qs.searchKeys)
	for _, search := range qs.searchKeys {
		width = maxInt(width, len(search))
	}
	width = minInt(width+10, maxX-4)
	height := minInt(2*lines+4, maxY-3)
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	maxY = y0 + height

	if (width <= 3) || (height < 4) {
		// Handle case where the viewport is too small to show a full frame
		return nil
	}

	v, err := g.SetView(querySelectView, x0, y0, x0+width, y0+height, 0)
	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		v.Frame = true
	}

	v.Title = "Quick Search"

	y0 += 3
	for i, search := range qs.searchKeys {
		if i < qs.scroll {
			continue
		}
		viewname := fmt.Sprintf("queryedit%d", i)
		v, err := g.SetView(viewname, x0+2, y0, x0+2+width-4, y0+2, 0)
		if err != nil && !gocui.IsUnknownView(err) {
			return err
		}
		qs.childViews = append(qs.childViews, viewname)
		v.Frame = i == qs.selected
		v.Clear()

		text := fmt.Sprintf("%s", search)
		if len(text) >= width-8 {
			text = text[0:width-8] + "..."
		}

		_, _ = fmt.Fprint(v, text)

		y0 += 2
		if y0 >= maxY {
			break
		}
	}

	_, _ = fmt.Fprint(v, querySelectHelp.Render(maxX))
	if _, err := g.SetCurrentView(querySelectView); err != nil {
		return err
	}
	return nil
}

func (qs *querySelect) disable(g *gocui.Gui) error {
	for _, view := range qs.childViews {
		if err := g.DeleteView(view); err != nil && !gocui.IsUnknownView(err) {
			return err
		}
	}
	return nil
}

func (qs *querySelect) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	if qs.selected < 0 {
		return nil
	}

	qs.selected = maxInt(0, qs.selected-1)
	qs.scroll = maxInt(0, qs.selected-len(qs.childViews)+1)
	return nil
}

func (qs *querySelect) selectNext(g *gocui.Gui, v *gocui.View) error {
	if qs.selected < 0 {
		return nil
	}

	qs.selected = minInt(len(qs.searchKeys)-1, qs.selected+1)
	qs.scroll = maxInt(0, qs.selected-len(qs.childViews)+1)
	return nil
}

func (qs *querySelect) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.bugTable)
}

func (qs *querySelect) selectAndReturn(g *gocui.Gui, v *gocui.View) error {
	if (qs.selected >= 0) && (qs.selected < len(qs.searchKeys)) {
		queryStr := qs.searches[qs.searchKeys[qs.selected]]
		updateQuery(ui.bugTable, queryStr)
	}
	return ui.activateWindow(ui.bugTable)
}
