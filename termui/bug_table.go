package termui

import (
	"bytes"
	"fmt"
	"strings"

	termtext "github.com/MichaelMure/go-term-text"
	"github.com/awesome-gocui/gocui"
	"github.com/dustin/go-humanize"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/query"
	"github.com/daedaleanai/git-ticket/util/colors"
)

const bugTableView = "bugTableView"
const bugTableHeaderView = "bugTableHeaderView"
const bugTableFooterView = "bugTableFooterView"
const bugTableInstructionView = "bugTableInstructionView"

const defaultRemote = "origin"
const defaultQuery = ""

var bugTableHelp = helpBar{
	{"q", "Quit"},
	{"s", "Search"},
	{"S", "Quick Search"},
	{"←↓↑→,hjkl", "Navigation"},
	{"↵", "Open bug"},
	{"n", "New bug"},
	{"i", "Pull"},
	{"o", "Push"},
}

type bugTable struct {
	repo         *cache.RepoCache
	queryStr     string
	query        *query.Query
	allIds       []entity.Id
	excerpts     []*cache.BugExcerpt
	pageCursor   int
	selectCursor int
	searches     map[string]string
}

func newBugTable(c *cache.RepoCache) *bugTable {

	searches, err := c.GetSearches()
	if err != nil {
		panic(err)
	}

	q, err := query.Parse(defaultQuery)
	if err != nil {
		panic(err)
	}

	return &bugTable{
		repo:         c,
		query:        q,
		queryStr:     defaultQuery,
		pageCursor:   0,
		selectCursor: 0,
		searches:     searches,
	}
}

func (bt *bugTable) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if maxY < 4 {
		// window too small !
		return nil
	}

	v, err := g.SetView(bugTableHeaderView, -1, -1, maxX, 1, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderHeader(v, maxX)

	v, err = g.SetView(bugTableView, -1, 0, maxX, maxY-3, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		v.Frame = false
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack
	}

	viewWidth, viewHeight := v.Size()
	err = bt.paginate(viewHeight)
	if err != nil {
		return err
	}

	err = bt.cursorClamp(v)
	if err != nil {
		return err
	}

	v.Clear()
	bt.render(v, viewWidth)

	v, err = g.SetView(bugTableFooterView, -1, maxY-4, maxX, maxY, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		v.Frame = false
	}

	v.Clear()
	bt.renderFooter(v, maxX)

	v, err = g.SetView(bugTableInstructionView, -1, maxY-2, maxX, maxY, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		v.Frame = false
		v.FgColor = gocui.ColorWhite
	}
	v.Clear()
	bt.renderHelp(v, maxX)

	_, err = g.SetCurrentView(bugTableView)
	return err
}

func (bt *bugTable) keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding(bugTableView, 'q', gocui.ModNone, quit); err != nil {
		return err
	}

	// Down
	if err := g.SetKeybinding(bugTableView, 'j', gocui.ModNone,
		bt.cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowDown, gocui.ModNone,
		bt.cursorDown); err != nil {
		return err
	}
	// Up
	if err := g.SetKeybinding(bugTableView, 'k', gocui.ModNone,
		bt.cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowUp, gocui.ModNone,
		bt.cursorUp); err != nil {
		return err
	}

	// Previous page
	if err := g.SetKeybinding(bugTableView, 'h', gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowLeft, gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyPgup, gocui.ModNone,
		bt.previousPage); err != nil {
		return err
	}
	// Next page
	if err := g.SetKeybinding(bugTableView, 'l', gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyArrowRight, gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}
	if err := g.SetKeybinding(bugTableView, gocui.KeyPgdn, gocui.ModNone,
		bt.nextPage); err != nil {
		return err
	}

	// New bug
	if err := g.SetKeybinding(bugTableView, 'n', gocui.ModNone,
		bt.newBug); err != nil {
		return err
	}

	// Open bug
	if err := g.SetKeybinding(bugTableView, gocui.KeyEnter, gocui.ModNone,
		bt.openBug); err != nil {
		return err
	}

	// Pull
	if err := g.SetKeybinding(bugTableView, 'i', gocui.ModNone,
		bt.pull); err != nil {
		return err
	}

	// Push
	if err := g.SetKeybinding(bugTableView, 'o', gocui.ModNone,
		bt.push); err != nil {
		return err
	}

	// Query
	if err := g.SetKeybinding(bugTableView, 's', gocui.ModNone,
		bt.changeQuery); err != nil {
		return err
	}

	// Quick Search
	if err := g.SetKeybinding(bugTableView, 'S', gocui.ModNone,
		bt.querySelect); err != nil {
		return err
	}

	return nil
}

func (bt *bugTable) disable(g *gocui.Gui) error {
	if err := g.DeleteView(bugTableView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	if err := g.DeleteView(bugTableHeaderView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	if err := g.DeleteView(bugTableFooterView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	if err := g.DeleteView(bugTableInstructionView); err != nil && !gocui.IsUnknownView(err) {
		return err
	}
	return nil
}

func (bt *bugTable) paginate(max int) error {
	bt.allIds = bt.repo.QueryBugs(bt.query)

	return bt.doPaginate(max)
}

func (bt *bugTable) doPaginate(max int) error {
	// clamp the cursor
	bt.pageCursor = maxInt(bt.pageCursor, 0)
	bt.pageCursor = minInt(bt.pageCursor, len(bt.allIds))

	nb := minInt(len(bt.allIds)-bt.pageCursor, max)

	if nb < 0 {
		bt.excerpts = []*cache.BugExcerpt{}
		return nil
	}

	// slice the data
	ids := bt.allIds[bt.pageCursor : bt.pageCursor+nb]

	bt.excerpts = make([]*cache.BugExcerpt, len(ids))

	for i, id := range ids {
		excerpt, err := bt.repo.ResolveBugExcerpt(id)
		if err != nil {
			return err
		}

		bt.excerpts[i] = excerpt
	}

	return nil
}

func (bt *bugTable) getTableLength() int {
	return len(bt.excerpts)
}

func (bt *bugTable) getColumnWidths(maxX int) map[string]int {
	m := make(map[string]int)
	m["id"] = 7
	m["status"] = 6

	left := maxX - 5 - m["id"] - m["status"]

	m["comments"] = 3
	left -= m["comments"]
	m["lastEdit"] = 14
	left -= m["lastEdit"]

	m["author"] = minInt(maxInt(left/3, 15), 10+left/8)
	m["title"] = maxInt(left-m["author"], 10)

	return m
}

func (bt *bugTable) render(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	for _, excerpt := range bt.excerpts {
		summaryTxt := fmt.Sprintf("%3d", excerpt.LenComments-1)
		if excerpt.LenComments-1 <= 0 {
			summaryTxt = ""
		}
		if excerpt.LenComments-1 > 999 {
			summaryTxt = "  ∞"
		}

		var labelsTxt strings.Builder
		for _, l := range excerpt.Labels {
			labelsTxt.WriteString(" ")
			lc256 := l.Color().Term256()
			labelsTxt.WriteString(lc256.Escape())
			labelsTxt.WriteString("◼")
			labelsTxt.WriteString(lc256.Unescape())
		}

		var authorDisplayName string
		if excerpt.AuthorId != "" {
			author, err := bt.repo.ResolveIdentityExcerpt(excerpt.AuthorId)
			if err != nil {
				panic(err)
			}
			authorDisplayName = author.DisplayName()
		} else {
			authorDisplayName = excerpt.LegacyAuthor.DisplayName()
		}

		lastEditTime := excerpt.EditTime()

		id := termtext.LeftPadMaxLine(excerpt.Id.Human(), columnWidths["id"], 0)
		status := termtext.LeftPadMaxLine(excerpt.Status.String(), columnWidths["status"], 0)
		labels := termtext.TruncateMax(labelsTxt.String(), minInt(columnWidths["title"]-2, 10))
		title := termtext.LeftPadMaxLine(strings.TrimSpace(excerpt.Title), columnWidths["title"]-termtext.Len(labels), 0)
		author := termtext.LeftPadMaxLine(authorDisplayName, columnWidths["author"], 0)
		comments := termtext.LeftPadMaxLine(summaryTxt, columnWidths["comments"], 0)
		lastEdit := termtext.LeftPadMaxLine(humanize.Time(lastEditTime), columnWidths["lastEdit"], 1)

		_, _ = fmt.Fprintf(v, "%s %s %s%s %s %s %s\n",
			colors.Cyan(id),
			colors.Yellow(status),
			title,
			labels,
			colors.Magenta(author),
			comments,
			lastEdit,
		)
	}

	_ = v.SetHighlight(bt.selectCursor, true)
}

func (bt *bugTable) renderHeader(v *gocui.View, maxX int) {
	columnWidths := bt.getColumnWidths(maxX)

	id := termtext.LeftPadMaxLine("ID", columnWidths["id"], 0)
	status := termtext.LeftPadMaxLine("STATUS", columnWidths["status"], 0)
	title := termtext.LeftPadMaxLine("TITLE", columnWidths["title"], 0)
	author := termtext.LeftPadMaxLine("AUTHOR", columnWidths["author"], 0)
	comments := termtext.LeftPadMaxLine("CMT", columnWidths["comments"], 0)
	lastEdit := termtext.LeftPadMaxLine("LAST EDIT", columnWidths["lastEdit"], 1)

	_, _ = fmt.Fprintf(v, "%s %s %s %s %s %s\n", id, status, title, author, comments, lastEdit)
}

func (bt *bugTable) renderFooter(v *gocui.View, maxX int) {
	_, _ = fmt.Fprintf(v, " \nShowing %d of %d bugs", len(bt.excerpts), len(bt.allIds))
}

func (bt *bugTable) renderHelp(v *gocui.View, maxX int) {
	_, _ = fmt.Fprint(v, bugTableHelp.Render(maxX))
}

func (bt *bugTable) cursorDown(g *gocui.Gui, v *gocui.View) error {
	// If we are at the bottom of the page, switch to the next one.
	if bt.selectCursor+1 > bt.getTableLength()-1 {
		_, max := v.Size()

		if bt.pageCursor+max >= len(bt.allIds) {
			return nil
		}

		bt.pageCursor += max
		bt.selectCursor = 0

		return bt.doPaginate(max)
	}

	bt.selectCursor = minInt(bt.selectCursor+1, bt.getTableLength()-1)

	return nil
}

func (bt *bugTable) cursorUp(g *gocui.Gui, v *gocui.View) error {
	// If we are at the top of the page, switch to the previous one.
	if bt.selectCursor-1 < 0 {
		_, max := v.Size()

		if bt.pageCursor == 0 {
			return nil
		}

		bt.pageCursor = maxInt(0, bt.pageCursor-max)
		bt.selectCursor = max - 1

		return bt.doPaginate(max)
	}

	bt.selectCursor = maxInt(bt.selectCursor-1, 0)

	return nil
}

func (bt *bugTable) cursorClamp(v *gocui.View) error {
	y := bt.selectCursor

	y = minInt(y, bt.getTableLength()-1)
	y = maxInt(y, 0)

	bt.selectCursor = y

	return nil
}

func (bt *bugTable) nextPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()

	if bt.pageCursor+max >= len(bt.allIds) {
		return nil
	}

	bt.pageCursor += max

	return bt.doPaginate(max)
}

func (bt *bugTable) previousPage(g *gocui.Gui, v *gocui.View) error {
	_, max := v.Size()

	if bt.pageCursor == 0 {
		return nil
	}

	bt.pageCursor = maxInt(0, bt.pageCursor-max)

	return bt.doPaginate(max)
}

func (bt *bugTable) newBug(g *gocui.Gui, v *gocui.View) error {
	return newBugWithEditor(bt.repo)
}

func (bt *bugTable) openBug(g *gocui.Gui, v *gocui.View) error {
	if len(bt.excerpts) == 0 {
		// There are no open bugs, just do nothing
		return nil
	}
	id := bt.excerpts[bt.selectCursor].Id
	b, err := bt.repo.ResolveBug(id)
	if err != nil {
		return err
	}
	ui.showBug.SetBug(b)
	return ui.activateWindow(ui.showBug)
}

func (bt *bugTable) pull(g *gocui.Gui, v *gocui.View) error {
	ui.msgPopup.Activate("Pull from remote "+defaultRemote, "...")

	go func() {
		stdout, err := bt.repo.Fetch(defaultRemote)

		if err != nil {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
				return nil
			})
		} else {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.UpdateMessage(stdout)
				return nil
			})
		}

		var buffer bytes.Buffer
		beginLine := ""

		for result := range bt.repo.MergeAll(defaultRemote) {
			if result.Status == entity.MergeStatusNothing {
				continue
			}

			if result.Err != nil {
				g.Update(func(gui *gocui.Gui) error {
					ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
					return nil
				})
			} else {
				_, _ = fmt.Fprintf(&buffer, "%s%s: %s",
					beginLine, colors.Cyan(result.Entity.Id().Human()), result,
				)

				beginLine = "\n"

				g.Update(func(gui *gocui.Gui) error {
					ui.msgPopup.UpdateMessage(buffer.String())
					return nil
				})
			}
		}

		_, _ = fmt.Fprintf(&buffer, "%sdone", beginLine)

		g.Update(func(gui *gocui.Gui) error {
			ui.msgPopup.UpdateMessage(buffer.String())
			return nil
		})

	}()

	return nil
}

func (bt *bugTable) push(g *gocui.Gui, v *gocui.View) error {
	ui.msgPopup.Activate("Push to remote "+defaultRemote, "...")

	go func() {
		// TODO: make the remote configurable
		out := new(bytes.Buffer)
		err := bt.repo.Push(defaultRemote, out)

		if err != nil {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
				return nil
			})
		} else {
			g.Update(func(gui *gocui.Gui) error {
				ui.msgPopup.UpdateMessage(out.String())
				return nil
			})
		}
	}()

	return nil
}

func (bt *bugTable) changeQuery(g *gocui.Gui, v *gocui.View) error {
	return editQueryWithEditor(bt)
}

func (bt *bugTable) querySelect(g *gocui.Gui, v *gocui.View) error {
	if len(bt.searches) > 0 {
		g.Update(func(gui *gocui.Gui) error {
			ui.querySelect.SetSearches(bt.searches)
			return nil
		})
		return ui.activateWindow(ui.querySelect)
	}
	return nil
}
