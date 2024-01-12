package termui

import (
	"fmt"

	"github.com/awesome-gocui/gocui"

	"github.com/daedaleanai/git-ticket/cache"
)

const timelineView = "timelineView"
const timelineInstructionsView = "timelineInstructionsView"

var timelineHelp = helpBar{
	{"q", "Quit"},
	{"↓↑,jk", "Nav"},
}

type timeline struct {
	bug      *cache.BugCache
	selected int
}

func newTimeline() *timeline {
	return &timeline{}
}

func (tl *timeline) SetBug(bug *cache.BugCache) {
	tl.bug = bug
	tl.selected = 0
}

func (tl *timeline) keybindings(g *gocui.Gui) error {
	// Abort / Quit
	if err := g.SetKeybinding(timelineView, gocui.KeyEsc, gocui.ModNone, tl.abort); err != nil {
		return err
	}
	if err := g.SetKeybinding(timelineView, 'q', gocui.ModNone, tl.abort); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(timelineView, gocui.KeyArrowUp, gocui.ModNone, tl.selectPrevious); err != nil {
		return err
	}
	if err := g.SetKeybinding(timelineView, 'k', gocui.ModNone, tl.selectPrevious); err != nil {
		return err
	}
	// Down
	if err := g.SetKeybinding(timelineView, gocui.KeyArrowDown, gocui.ModNone, tl.selectNext); err != nil {
		return err
	}
	if err := g.SetKeybinding(timelineView, 'j', gocui.ModNone, tl.selectNext); err != nil {
		return err
	}
	return nil
}

func (tl *timeline) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	v, err := g.SetView(timelineView, 0, 0, maxX, maxY-2, 0)
	if err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	v.Frame = false
	v.Clear()

	for i, timeline := range tl.bug.Snapshot().Timeline {
		if i < tl.selected {
			continue
		}

		_, _ = fmt.Fprintln(v, timeline)
	}

	v, err = g.SetView(timelineInstructionsView, -1, maxY-2, maxX, maxY, 0)
	if err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	v.Frame = false
	v.FgColor = gocui.ColorWhite
	v.Clear()

	_, _ = fmt.Fprint(v, timelineHelp.Render(maxX))

	if _, err = g.SetViewOnTop(timelineInstructionsView); err != nil {
		return err
	}
	if _, err := g.SetCurrentView(timelineView); err != nil {
		return err
	}
	return nil
}

func (tl *timeline) disable(g *gocui.Gui) error {
	if err := g.DeleteView(timelineView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	if err := g.DeleteView(timelineInstructionsView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	return nil
}

func (tl *timeline) selectPrevious(g *gocui.Gui, v *gocui.View) error {
	tl.selected = maxInt(0, tl.selected-1)
	return nil
}

func (tl *timeline) selectNext(g *gocui.Gui, v *gocui.View) error {
	tl.selected = minInt(len(tl.bug.Snapshot().Timeline)-1, tl.selected+1)
	return nil
}

func (tl *timeline) abort(g *gocui.Gui, v *gocui.View) error {
	return ui.activateWindow(ui.showBug)
}
