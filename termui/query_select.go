package termui

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

const querySelectView = "querySelectView"

var querySelectHelp = helpBar{
	{"q", "Close"},
	{"↓↑,jk", "Nav"},
	{"↵", "Select query"},
}

type querySelect struct {
	searches    []Search
	selected    int
	scroll      int
	childViews  []string
	cnt         int
}

func newQuerySelect() *querySelect {
	return &querySelect{}
}

func (ls *querySelect) SetSearches(configuration *Configuration) {
	ls.searches = configuration.Searches
	ls.selected = 0 
	ls.scroll = 0
	ls.cnt = ls.cnt + 1
}

func (ls *querySelect) keybindings(g *gocui.Gui) error {
	// Abort
	if err := g.SetKeybinding(querySelectView, gocui.KeyEsc, gocui.ModNone, ls.abort); err != nil {
		return err
	}
	// Save and return
	if err := g.SetKeybinding(querySelectView, 'q', gocui.ModNone, ls.abort); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(querySelectView, gocui.KeyArrowUp, gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'k', gocui.ModNone, ls.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(querySelectView, gocui.KeyArrowDown, gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'j', gocui.ModNone, ls.selectNext); err != nil {
		return err
	}
	// Select
	if err := g.SetKeybinding(querySelectView, gocui.KeySpace, gocui.ModNone, ls.selectAndReturn); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, 'x', gocui.ModNone, ls.selectAndReturn); err != nil {
		return err
	}
	if err := g.SetKeybinding(querySelectView, gocui.KeyEnter, gocui.ModNone, ls.selectAndReturn); err != nil {
		return err
	}
	return nil
}

func (ls *querySelect) layout(g *gocui.Gui) error {
	// Clean up child views to support scrolling
	for _, view := range ls.childViews {
		if err := g.DeleteView(view); err != nil && !gocui.IsUnknownView(err) {
			return err
		}
	}
	ls.childViews = nil

	maxX, maxY := g.Size()
	width := minInt(10, maxX) 
	lines := len(ls.searches)
	for _, search := range ls.searches {
		width = maxInt(width, len(search.Name) + 3 + len(search.Search))
	}
	width = minInt(width + 10, maxX - 4)
	height := minInt(2*lines+2, maxY-3)
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

	y0 += 1
	for i, search := range ls.searches {
		if i < ls.scroll {
			continue
		}
		viewname := fmt.Sprintf("queryedit%d", i)
		v, err := g.SetView(viewname, x0+2, y0, x0+2+width-4, y0+2, 0)
		if err != nil && !gocui.IsUnknownView(err) {
			return err
		}
		ls.childViews = append(ls.childViews, viewname)
		v.Frame = i == ls.selected
		v.Clear()

		text := fmt.Sprintf("%s : %s", search.Name, search.Search)
		if len(text) >= width-8 {
			text = text[0:width-8] + "..."
		}

		_, _ = fmt.Fprint(v, text) 

		y0 += 2
		if y0 >= maxY {
			//break
		}
	}

	if _, err := g.SetCurrentView(querySelectView); err != nil {
		return err
	}
	return nil
}

func (ls *querySelect) disable(g *gocui.Gui) error {
	for _, view := range ls.childViews {
		if err := g.DeleteView(view); err != nil && !gocui.IsUnknownView(err) {
			return err
		}
	}
	return nil
}

func (ls *querySelect) focusView(g *gocui.Gui) error {
	if ls.selected < 0 {
		return nil
	}

	ls.scroll = maxInt(0, ls.selected - len(ls.childViews) + 1)

	return nil
}

func (ls *querySelect) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	if ls.selected < 0 {
		return nil
	}

	ls.selected = maxInt(0, ls.selected-1)
	return ls.focusView(g)
}

func (ls *querySelect) selectNext(g *gocui.Gui, v *gocui.View) error {
	if ls.selected < 0 {
		return nil
	}

	ls.selected = minInt(len(ls.searches)-1, ls.selected+1)
	return ls.focusView(g)
}

func (ls *querySelect) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.bugTable)
}

func (ls *querySelect) selectAndReturn(g *gocui.Gui, v *gocui.View) error {
	if ((ls.selected >= 0) && (ls.selected < len(ls.searches))) { 
		queryStr := ls.searches[ls.selected].Search
		updateQuery(ui.bugTable, queryStr)
	}
	return ui.activateWindow(ui.bugTable)
}
