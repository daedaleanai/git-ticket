package webui

import (
	"fmt"
	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/gorilla/sessions"
	"net/http"
	"net/url"
)

func handleCreateTicket(repo *cache.RepoCache, w http.ResponseWriter, r *http.Request) error {
	session := r.Context().Value(ddlnContextKeySession).(*sessions.Session)
	defer session.Save(r, w)

	var err error
	var validationErrors = make(map[string]*invalidRequestError)
	var formData url.Values

	if r.Method == http.MethodPost {
		if err = r.ParseForm(); err != nil {
			return fmt.Errorf("failed to parse form data: %w", err)
		}

		var action *CreateTicketAction
		formData = r.Form
		action, validationErrors, err := createTicketFromFormData(formData, repo)

		if len(validationErrors) == 0 && err == nil {
			ticket, _, err := repo.NewBug(cache.NewBugOpts{
				Title:    action.Title,
				Message:  action.Message,
				Workflow: action.Workflow,
				Assignee: action.AssignedTo,
				Repo:     fmt.Sprintf("%s%s", bug.RepoPrefix, action.Repo),
			})
			if err != nil {
				session.AddFlash(fmt.Sprintf("Failed to create ticket: %s", err.Error()))
			} else {
				http.Redirect(w, r, fmt.Sprintf("/ticket/%s/", ticket.Id()), http.StatusSeeOther)
				return nil
			}
		}

		if err != nil {
			session.AddFlash(fmt.Sprintf("Failed to create ticket: %s", err.Error()))
		}
	}

	flashes := session.Flashes()

	repoLabels, err := repo.ListRepoLabels()
	if err != nil {
		return err
	}

	data := struct {
		SideBar          SideBarData
		WorkflowLabels   []bug.Label
		RepoLabels       []string
		ValidationErrors map[string]*invalidRequestError
		FlashErrors      []interface{}
		FormData         url.Values
		UserOptions      []*cache.IdentityExcerpt
	}{
		SideBar: SideBarData{
			BookmarkGroups: webUiConfig.BookmarkGroups,
			ColorKey:       map[string]string{},
		},
		WorkflowLabels:   bug.GetWorkflowLabels(),
		ValidationErrors: validationErrors,
		RepoLabels:       repoLabels,
		FormData:         formData,
		FlashErrors:      flashes,
		UserOptions:      repo.AllIdentityExcerpts(),
	}

	return renderTemplate(w, "create.html", data)
}

type CreateTicketAction struct {
	Title      string
	Message    string
	Workflow   string
	Repo       string
	AssignedTo identity.Interface
	Ccb        []string
	Labels     []string
	Checklists []string
}

const keyTitle = "title"
const keyWorkflow = "workflow"
const keyRepo = "repo"
const keyAssignee = "assignee"
const keyMessage = "description"

func createTicketFromFormData(f url.Values, c *cache.RepoCache) (*CreateTicketAction, map[string]*invalidRequestError, error) {
	required := [4]string{keyTitle, keyWorkflow, keyRepo, keyMessage}
	var validationErrors = make(map[string]*invalidRequestError)

	for k := range f {
		// Unset any empty values
		if len(f.Get(k)) == 0 {
			f.Del(k)
		}
	}

	for _, v := range required {
		if !f.Has(v) {
			validationErrors[v] = &invalidRequestError{msg: fmt.Sprintf("%s is required", v)}
		}
	}

	if f.Has(keyWorkflow) && !isValidWorkflow(f.Get(keyWorkflow)) {
		l := bug.Label(f.Get(keyWorkflow))
		validationErrors[keyWorkflow] = &invalidRequestError{msg: fmt.Sprintf("%s is not a valid workflow", l.WorkflowName())}
	}

	var assignee identity.Interface
	if f.Has(keyAssignee) {
		var err error
		assignee, err = resolveIdentityFromFormValue(c, f.Get(keyAssignee), validationErrors)
		if err != nil {
			return nil, validationErrors, err
		}
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors, nil
	}

	return &CreateTicketAction{
		Title:      f.Get(keyTitle),
		Message:    f.Get(keyMessage),
		Workflow:   f.Get(keyWorkflow),
		Repo:       f.Get(keyRepo),
		AssignedTo: assignee,
		Ccb:        nil,
		Labels:     nil,
		Checklists: nil,
	}, nil, nil
}

func resolveIdentityFromFormValue(
	c *cache.RepoCache,
	value string, validationErrors map[string]*invalidRequestError,
) (identity.Interface, error) {
	id, err := c.ResolveIdentity(entity.Id(value))

	if err == identity.ErrIdentityNotExist {
		validationErrors[keyAssignee] = &invalidRequestError{msg: fmt.Sprintf("user %s does not exist", value)}
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return id.Identity, nil
}

func isValidWorkflow(s string) bool {
	for _, l := range bug.GetWorkflowLabels() {
		label := bug.Label(s)
		if l == label {
			return true
		}
	}

	return false
}
