// Package termui contains the interactive terminal UI
package termui

import (
	"fmt"
	"sort"

	"github.com/awesome-gocui/gocui"
	"github.com/pkg/errors"

	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/config"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/input"
	"github.com/daedaleanai/git-ticket/query"
)

var errTerminateMainloop = errors.New("terminate gocui mainloop")
var noneString string = "<None>"
var exitString string = "<Exit>"

type termUI struct {
	g      *gocui.Gui
	gError chan error
	cache  *cache.RepoCache

	activeWindow window

	bugTable    *bugTable
	showBug     *showBug
	labelSelect *labelSelect
	timeline    *timeline
	msgPopup    *msgPopup
	inputPopup  *inputPopup
}

func (tui *termUI) activateWindow(window window) error {
	if err := tui.activeWindow.disable(tui.g); err != nil {
		return err
	}

	tui.activeWindow = window

	return nil
}

var ui *termUI

type window interface {
	keybindings(g *gocui.Gui) error
	layout(g *gocui.Gui) error
	disable(g *gocui.Gui) error
}

// Run will launch the termUI in the terminal
func Run(cache *cache.RepoCache) error {
	ui = &termUI{
		gError:      make(chan error, 1),
		cache:       cache,
		bugTable:    newBugTable(cache),
		showBug:     newShowBug(cache),
		labelSelect: newLabelSelect(),
		timeline:    newTimeline(),
		msgPopup:    newMsgPopup(),
		inputPopup:  newInputPopup(),
	}

	ui.activeWindow = ui.bugTable

	initGui(nil)

	err := <-ui.gError

	type errorStack interface {
		ErrorStack() string
	}

	if err != nil && err != gocui.ErrQuit {
		if e, ok := err.(errorStack); ok {
			fmt.Println(e.ErrorStack())
		}
		return err
	}

	return nil
}

func initGui(action func(ui *termUI) error) {
	g, err := gocui.NewGui(gocui.Output256, false)

	if err != nil {
		ui.gError <- err
		return
	}

	ui.g = g

	ui.g.SetManagerFunc(layout)

	ui.g.InputEsc = true

	err = keybindings(ui.g)

	if err != nil {
		ui.g.Close()
		ui.g = nil
		ui.gError <- err
		return
	}

	if action != nil {
		err = action(ui)
		if err != nil {
			ui.g.Close()
			ui.g = nil
			ui.gError <- err
			return
		}
	}

	err = g.MainLoop()

	if err != nil && err != errTerminateMainloop {
		if ui.g != nil {
			ui.g.Close()
		}
		ui.gError <- err
	}
}

func layout(g *gocui.Gui) error {
	g.Cursor = false

	if err := ui.activeWindow.layout(g); err != nil {
		return err
	}

	if err := ui.msgPopup.layout(g); err != nil {
		return err
	}

	if err := ui.inputPopup.layout(g); err != nil {
		return err
	}

	return nil
}

func keybindings(g *gocui.Gui) error {
	// Quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := ui.bugTable.keybindings(g); err != nil {
		return err
	}

	if err := ui.showBug.keybindings(g); err != nil {
		return err
	}

	if err := ui.labelSelect.keybindings(g); err != nil {
		return err
	}

	if err := ui.timeline.keybindings(g); err != nil {
		return err
	}

	if err := ui.msgPopup.keybindings(g); err != nil {
		return err
	}

	if err := ui.inputPopup.keybindings(g); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

type completionCallback func(interface{}, error) error
type uiAction func(interface{}, completionCallback) error

func generateWorkflowQuery() uiAction {
	return func(arg interface{}, cb completionCallback) error {
		workflowLabels := bug.GetWorkflowLabels()
		workflows := make([]string, 0, len(workflowLabels))

		for _, k := range workflowLabels {
			workflows = append(workflows, string(k))
		}

		c := ui.inputPopup.ActivateWithContent("[↓↑] Select workflows", workflows)
		go func() {
			selectedWorkflow := <-c

			bugOpts := arg.(*cache.NewBugOpts)
			bugOpts.Workflow = selectedWorkflow

			cb(bugOpts, nil)
		}()

		return nil
	}
}

func generateRepoQuery(repo *cache.RepoCache) uiAction {
	return func(arg interface{}, cb completionCallback) error {
		var repoLabels []string
		err := repo.DoWithLockedConfigCache(func(config *config.ConfigCache) error {
			inner, err := config.ListLabelsWithNamespace("repo")
			repoLabels = inner
			return err
		})
		if err != nil {
			return err
		}
		sort.Slice(repoLabels, func(i, j int) bool { return repoLabels[i] < repoLabels[j] })

		c := ui.inputPopup.ActivateWithContent("[↓↑] Select repository", repoLabels)
		go func() {
			selectedRepo := <-c

			bugOpts := arg.(*cache.NewBugOpts)
			bugOpts.Repo = bug.RepoPrefix + selectedRepo

			cb(bugOpts, nil)
		}()

		return nil
	}
}

func generateMilestoneQuery(repo *cache.RepoCache) uiAction {
	return func(arg interface{}, cb completionCallback) error {
		var milestoneLabels []string
		err := repo.DoWithLockedConfigCache(func(config *config.ConfigCache) error {
			inner, err := config.ListLabelsWithNamespace("milestone")
			milestoneLabels = inner
			return err
		})
		if err != nil {
			return err
		}
		sort.Slice(milestoneLabels, func(i, j int) bool { return milestoneLabels[i] < milestoneLabels[j] })

		milestoneLabels = append([]string{noneString}, milestoneLabels...)

		c := ui.inputPopup.ActivateWithContent("[↓↑] Select milestone", milestoneLabels)
		go func() {
			selectedMilestone := <-c

			bugOpts := arg.(*cache.NewBugOpts)
			if selectedMilestone == noneString {
				bugOpts.Milestone = ""
			} else {
				bugOpts.Milestone = bug.MilestonePrefix + selectedMilestone
			}

			cb(bugOpts, nil)
		}()

		return nil
	}
}

var errNoMoreRepeats = errors.New("No more repeats were requested")

func generateImpactQuery(repo *cache.RepoCache) uiAction {
	return func(arg interface{}, cb completionCallback) error {
		var allImpactLabels []string
		err := repo.DoWithLockedConfigCache(func(config *config.ConfigCache) error {
			inner, err := config.ListLabelsWithNamespace("impact")
			allImpactLabels = inner
			return err
		})
		if err != nil {
			return err
		}

		bugOpts := arg.(*cache.NewBugOpts)

		selectedImpactLabels := make(map[string]struct{})
		for _, impact := range bugOpts.Impact {
			selectedImpactLabels[impact] = struct{}{}
		}

		var impactLabels []string
		for _, label := range allImpactLabels {
			if _, ok := selectedImpactLabels[bug.ImpactPrefix+label]; !ok {
				impactLabels = append(impactLabels, label)
			}
		}

		sort.Slice(impactLabels, func(i, j int) bool { return impactLabels[i] < impactLabels[j] })

		impactLabels = append([]string{exitString}, impactLabels...)

		c := ui.inputPopup.ActivateWithContent("[↓↑] Select impact", impactLabels)
		go func() {
			selectedImpact := <-c

			if selectedImpact != exitString {
				bugOpts.Impact = append(bugOpts.Impact, bug.ImpactPrefix+selectedImpact)
				cb(bugOpts, nil)
			} else {
				cb(bugOpts, errNoMoreRepeats)
			}

		}()

		return nil
	}
}

func generateScopeQuery(repo *cache.RepoCache) uiAction {
	return func(arg interface{}, cb completionCallback) error {
		var allScopeLabels []string
		err := repo.DoWithLockedConfigCache(func(config *config.ConfigCache) error {
			inner, err := config.ListLabelsWithNamespace("scope")
			allScopeLabels = inner
			return err
		})
		if err != nil {
			return err
		}

		bugOpts := arg.(*cache.NewBugOpts)

		selectedScopeLabels := make(map[string]struct{})
		for _, scope := range bugOpts.Scope {
			selectedScopeLabels[scope] = struct{}{}
		}

		var scopeLabels []string
		for _, label := range allScopeLabels {
			if _, ok := selectedScopeLabels[bug.ScopePrefix+label]; !ok {
				scopeLabels = append(scopeLabels, label)
			}
		}

		sort.Slice(scopeLabels, func(i, j int) bool { return scopeLabels[i] < scopeLabels[j] })

		scopeLabels = append([]string{exitString}, scopeLabels...)

		c := ui.inputPopup.ActivateWithContent("[↓↑] Select scope", scopeLabels)
		go func() {
			selectedScope := <-c

			if selectedScope != exitString {
				bugOpts.Scope = append(bugOpts.Scope, bug.ScopePrefix+selectedScope)
				cb(bugOpts, nil)
			} else {
				cb(bugOpts, errNoMoreRepeats)
			}
		}()

		return nil
	}
}

func repeatAction(inner uiAction) uiAction {
	return func(arg interface{}, cb completionCallback) error {
		var innerCb func(result interface{}, err error) error
		innerCb = func(result interface{}, err error) error {
			if err == errNoMoreRepeats {
				return cb(result, nil)
			}
			return inner(result, innerCb)
		}
		return inner(arg, innerCb)
	}
}

func generateCcbQuery(repo *cache.RepoCache) uiAction {
	type ccbArgs struct {
		primaryCCBPerTeam   map[string]entity.Id
		secondaryCCBPerTeam map[string]entity.Id
		labelMapping        config.LabelMapping
		ccbConfig           config.CcbConfig
		labels              []string
	}

	singleCcbQuery := func(arg interface{}, cb completionCallback) error {
		ccbArgs := arg.(*ccbArgs)

		handleCcbGroup := func(requiredTeams []string, ccbGroup map[string]entity.Id, otherGroup map[string]entity.Id, groupName string) (bool, error) {
			for _, teamName := range requiredTeams {
				if _, alreadyChosen := ccbGroup[teamName]; !alreadyChosen {

					var excludedMember *entity.Id = nil
					if secondaryCcbMember, ok := otherGroup[teamName]; ok {
						excludedMember = &secondaryCcbMember
					}

					ccbTeam, err := ccbArgs.ccbConfig.GetCcbTeam(teamName)
					if err != nil {
						return false, err
					}

					members := map[string]*cache.IdentityExcerpt{}
					var memberNames []string
					for _, member := range ccbTeam.Members {
						if excludedMember != nil && member.Id == *excludedMember {
							continue
						}

						id, err := repo.ResolveIdentityExcerpt(member.Id)
						if err != nil {
							return false, err
						}

						members[id.DisplayName()] = id
						memberNames = append(memberNames, id.DisplayName())
					}
					memberNames = append(memberNames, "<None>")

					message := fmt.Sprintf("[↓↑] Select CCB %s member for team %s", groupName, teamName)
					c := ui.inputPopup.ActivateWithContent(message, memberNames)

					go func() {
						selectedMember := <-c
						if member, ok := members[selectedMember]; ok {
							ccbGroup[teamName] = member.Id
						}

						cb(ccbArgs, nil)
					}()

					return true, nil
				}
			}
			return false, nil
		}

		// Take the first entry in labels that has a CCB mapping and process it
		for len(ccbArgs.labels) != 0 {
			label := ccbArgs.labels[0]
			if mapping, ok := ccbArgs.labelMapping[config.Label(label)]; ok {
				if triggeredAction, err := handleCcbGroup(mapping.PrimaryCcbTeams, ccbArgs.primaryCCBPerTeam, ccbArgs.secondaryCCBPerTeam, "primary"); err != nil || triggeredAction {
					return err
				}

				if triggeredAction, err := handleCcbGroup(mapping.SecondaryCcbTeams, ccbArgs.secondaryCCBPerTeam, ccbArgs.primaryCCBPerTeam, "secondary"); err != nil || triggeredAction {
					return err
				}
			}

			// Remove label
			ccbArgs.labels = ccbArgs.labels[1:]
		}

		return cb(ccbArgs, errNoMoreRepeats)
	}

	return func(arg interface{}, cb completionCallback) error {
		// The keys to these maps is the CCB team. The values are the identities selected for each team.
		// In principle there is no reason to have more than one person from a given team CCB'ing the change,
		// other than the fact that they may be different between primary and secondary, which is already handled
		// by having separate data structures for each.

		var labelMapping config.LabelMapping
		var ccbConfig config.CcbConfig
		_ = repo.DoWithLockedConfigCache(func(config *config.ConfigCache) error {
			ccbConfig = config.CcbConfig
			labelMapping = config.LabelMapping()
			return nil
		})

		bugOpts := arg.(*cache.NewBugOpts)
		labels := append([]string{bugOpts.Repo}, bugOpts.Impact...)

		innerArgs := &ccbArgs{
			primaryCCBPerTeam:   map[string]entity.Id{},
			secondaryCCBPerTeam: map[string]entity.Id{},
			labelMapping:        labelMapping,
			labels:              labels,
			ccbConfig:           ccbConfig,
		}

		var innerCb func(result interface{}, err error) error
		innerCb = func(result interface{}, err error) error {
			if err == errNoMoreRepeats {
				ccbArgs := result.(*ccbArgs)
				bugOpts.CcbMembers = map[bug.Status][]entity.Id{
					bug.VettedStatus:   make([]entity.Id, 0),
					bug.AcceptedStatus: make([]entity.Id, 0),
				}
				for _, member := range ccbArgs.primaryCCBPerTeam {
					bugOpts.CcbMembers[bug.VettedStatus] = append(bugOpts.CcbMembers[bug.VettedStatus], member)
					bugOpts.CcbMembers[bug.AcceptedStatus] = append(bugOpts.CcbMembers[bug.AcceptedStatus], member)
				}

				for _, member := range ccbArgs.secondaryCCBPerTeam {
					bugOpts.CcbMembers[bug.VettedStatus] = append(bugOpts.CcbMembers[bug.VettedStatus], member)
				}

				return cb(bugOpts, nil)
			}
			return singleCcbQuery(result, innerCb)
		}

		return singleCcbQuery(innerArgs, innerCb)
	}
}

func runChainedActions(arg interface{}, cb completionCallback, actions ...uiAction) error {
	if len(actions) == 0 {
		return cb(arg, nil)
	}

	actions[0](arg, func(result interface{}, err error) error {
		if err != nil {
			return cb(result, err)
		}

		return runChainedActions(result, cb, actions[1:]...)
	})

	return nil
}

func newBugWithEditor(repo *cache.RepoCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	title, message, err := input.BugCreateEditorInput(ui.cache, "", "")

	if err != nil && err != input.ErrEmptyTitle {
		return err
	}

	if err == input.ErrEmptyTitle {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty title, aborting.")
		initGui(nil)

		return errTerminateMainloop
	} else {
		initGui(func(ui *termUI) error {

			cb := func(result interface{}, err error) error {
				// We have filled all the bug options at this point
				newBugOpts := result.(*cache.NewBugOpts)

				labels := append([]string{newBugOpts.Repo}, newBugOpts.Impact...)
				selectedChecklists, _ := repo.FindChecklists(labels)
				newBugOpts.Checklists = selectedChecklists

				b, _, err := repo.NewBug(*newBugOpts)

				ui.g.Update(func(g *gocui.Gui) error {
					if err != nil {
						ui.msgPopup.Activate(msgPopupErrorTitle, fmt.Sprintf("Error creating ticket: %v", err))
						return nil
					}

					ui.showBug.SetBug(b)
					return ui.activateWindow(ui.showBug)
				})
				return nil
			}

			newBugOpts := &cache.NewBugOpts{Title: title, Message: message}

			return runChainedActions(newBugOpts, cb,
				generateWorkflowQuery(),
				generateRepoQuery(repo),
				generateMilestoneQuery(repo),
				repeatAction(generateImpactQuery(repo)),
				repeatAction(generateScopeQuery(repo)),
				generateCcbQuery(repo),
			)
		})

		return errTerminateMainloop
	}
}

func addCommentWithEditor(bug *cache.BugCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	message, err := input.BugCommentEditorInput(ui.cache, "")

	if err != nil && err != input.ErrEmptyMessage {
		return err
	}

	if err == input.ErrEmptyMessage {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty message, aborting.")
	} else {
		_, err := bug.AddComment(message)
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func editCommentWithEditor(bug *cache.BugCache, target entity.Id, preMessage string) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	message, err := input.BugCommentEditorInput(ui.cache, preMessage)
	if err != nil && err != input.ErrEmptyMessage {
		return err
	}

	if err == input.ErrEmptyMessage {
		// TODO: Allow comments to be deleted?
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty message, aborting.")
	} else if message == preMessage {
		ui.msgPopup.Activate(msgPopupErrorTitle, "No changes found, aborting.")
	} else {
		_, err := bug.EditComment(target, message)
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func reviewWithEditor(bug *cache.BugCache, checklist config.Checklist) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	clChange, err := input.ChecklistEditorInput(ui.cache, checklist, false)
	if err != nil {
		ui.msgPopup.Activate("", fmt.Sprintf("checklist not saved, re-execute command to continue editing: %s", err))
	} else if !clChange {
		ui.msgPopup.Activate("", "Checklists unchanged")
	} else {
		_, err := bug.SetChecklist(checklist)
		if err != nil {
			return err
		}
		ui.msgPopup.Activate("", checklist.Title+" updated")
	}

	initGui(nil)

	return errTerminateMainloop
}

func setTitleWithEditor(bug *cache.BugCache) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	snap := bug.Snapshot()

	title, err := input.BugTitleEditorInput(ui.cache, snap.Title)

	if err != nil && err != input.ErrEmptyTitle {
		return err
	}

	if err == input.ErrEmptyTitle {
		ui.msgPopup.Activate(msgPopupErrorTitle, "Empty title, aborting.")
	} else if title == snap.Title {
		ui.msgPopup.Activate(msgPopupErrorTitle, "No change, aborting.")
	} else {
		_, err := bug.SetTitle(title)
		if err != nil {
			return err
		}
	}

	initGui(nil)

	return errTerminateMainloop
}

func editQueryWithEditor(bt *bugTable) error {
	// This is somewhat hacky.
	// As there is no way to pause gocui, run the editor and restart gocui,
	// we have to stop it entirely and start a new one later.
	//
	// - an error channel is used to route the returned error of this new
	// 		instance into the original launch function
	// - a custom error (errTerminateMainloop) is used to terminate the original
	//		instance's mainLoop. This error is then filtered.

	ui.g.Close()
	ui.g = nil

	queryStr, err := input.QueryEditorInput(bt.repo, bt.queryStr)

	if err != nil {
		return err
	}

	bt.queryStr = queryStr

	parser, err := query.NewParser(queryStr)

	var q *query.CompiledQuery
	if err == nil {
		q, err = parser.Parse()
	}

	if err != nil {
		ui.msgPopup.Activate(msgPopupErrorTitle, err.Error())
	} else {
		bt.query = q
	}

	initGui(nil)

	return errTerminateMainloop
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}
