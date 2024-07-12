package webui

import (
	"errors"
	"fmt"
	"github.com/daedaleanai/git-ticket/bug"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	"github.com/daedaleanai/git-ticket/identity"
	http_webui "github.com/daedaleanai/git-ticket/webui/http"
	"github.com/daedaleanai/git-ticket/webui/session"
	"net/http"
	"net/url"
	"reflect"
)

func handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	repo := http_webui.LoadFromContext(r.Context(), &http_webui.ContextualRepoCache{}).(*http_webui.ContextualRepoCache).Repo
	bag := http_webui.LoadFromContext(r.Context(), &session.FlashMessageBag{}).(*session.FlashMessageBag)

	if r.Method == http.MethodPost {
		ticket, err := createTicket(r, repo)

		if err != nil {
			bag.AddMessage(session.NewError(fmt.Sprintf("Failed to create ticket: %s", err)))
		} else {
			bag.AddMessage(session.NewSuccess("Ticket created"))
			http.Redirect(w, r, fmt.Sprintf("/ticket/%s/", ticket.Id()), http.StatusSeeOther)
			return
		}
	}

	flashes := bag.Messages()

	repoLabels, err := repo.ListRepoLabels()
	if err != nil {
		http_webui.ErrorIntoResponse(err, w)
		return
	}

	data := struct {
		SideBar          SideBarData
		WorkflowLabels   []bug.Label
		RepoLabels       []string
		ValidationErrors map[string]session.FlashValidationError
		FlashErrors      []session.FlashMessage
		FormData         url.Values
		UserOptions      []*cache.IdentityExcerpt
	}{
		SideBar: SideBarData{
			BookmarkGroups: webUiConfig.BookmarkGroups,
			ColorKey:       map[string]string{},
		},
		WorkflowLabels:   bug.GetWorkflowLabels(),
		ValidationErrors: bag.ValidationErrors(),
		RepoLabels:       repoLabels,
		FormData:         r.Form,
		FlashErrors:      flashes,
		UserOptions:      repo.AllIdentityExcerpts(),
	}

	renderTemplate(w, "create.html", data)
}

func createTicket(r *http.Request, repo *cache.RepoCache) (*cache.BugCache, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	action := createTicketActionFromValues(r.Form).(CreateTicketAction)

	if !http_webui.IsValid(action, repo) {
		return nil, errors.New("ticket action is invalid")
	}

	assignee, err := action.getAssignee(repo)
	if err != nil {
		return nil, err
	}

	ticket, _, err := repo.NewBug(cache.NewBugOpts{
		Title:    action.Title,
		Message:  action.Message,
		Workflow: action.Workflow,
		Assignee: assignee,
		Repo:     fmt.Sprintf("%s%s", bug.RepoPrefix, action.Repo),
	})

	return ticket, err
}

func (a CreateTicketAction) getAssignee(repo *cache.RepoCache) (identity.Interface, error) {
	if a.AssignedTo == nil {
		return nil, nil
	}

	assignee, err := repo.ResolveIdentity(*a.AssignedTo)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve assignee: %w", err)
	}

	return assignee.Identity, nil
}

type CreateTicketAction struct {
	Title      string
	Message    string
	Workflow   string
	Repo       string
	AssignedTo *entity.Id
	Ccb        []string
	Labels     []string
	Checklists []string
}

func createTicketActionFromValues(values url.Values) http_webui.ValidatedPayload {
	for k := range values {
		// Unset any empty values, so we can evaluate with `values.Has`
		// Empty string values will always evaluate `values.Has` to `true` otherwise which doesn't make much sense.
		if len(values.Get(k)) == 0 {
			values.Del(k)
		}
	}

	c := CreateTicketAction{
		Title:    values.Get(keyTitle),
		Message:  values.Get(keyMessage),
		Workflow: values.Get(keyWorkflow),
		Repo:     values.Get(keyRepo),
	}

	if values.Has(keyAssignee) {
		id := entity.Id(values.Get(keyAssignee))
		c.AssignedTo = &id
	}

	return c
}

func (c CreateTicketAction) Validate(repo *cache.RepoCache) map[string]http_webui.ValidationError {
	required := [4]string{"Title", "Workflow", "Repo", "Message"}
	var validationErrors = make(map[string]http_webui.ValidationError)
	action := reflect.ValueOf(c)

	for _, v := range required {
		val := action.FieldByName(v)
		if val.String() == "" {
			validationErrors[v] = http_webui.ValidationError{Msg: fmt.Sprintf("%s is required", v)}
		}
	}

	if !isValidWorkflow(c.Workflow) {
		l := bug.Label(c.Workflow)
		validationErrors[keyWorkflow] = http_webui.ValidationError{Msg: fmt.Sprintf("%s is not a valid workflow", l.WorkflowName())}
	}

	if c.AssignedTo != nil {
		if _, err := repo.ResolveIdentity(*c.AssignedTo); err != nil {
			validationErrors[keyWorkflow] = http_webui.ValidationError{Msg: fmt.Sprintf("%s is not a valid user", c.AssignedTo)}
		}
	}

	return validationErrors
}

const keyTitle = "title"
const keyWorkflow = "workflow"
const keyRepo = "repo"
const keyAssignee = "assignee"
const keyMessage = "description"

func isValidWorkflow(s string) bool {
	for _, l := range bug.GetWorkflowLabels() {
		label := bug.Label(s)
		if l == label {
			return true
		}
	}

	return false
}
