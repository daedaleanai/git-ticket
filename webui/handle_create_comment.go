package webui

import (
	"fmt"
	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
	http_webui "github.com/daedaleanai/git-ticket/webui/http"
	"github.com/gorilla/mux"
	"net/http"
	"net/url"
)

type submitCommentAction struct {
	Ticket  string
	Comment string
}

func submitCommentFromFormData(ticketId string, f url.Values) (*submitCommentAction, error) {
	if !f.Has("comment") {
		return nil, &invalidRequestError{msg: "missing required field [comment]"}
	}

	return &submitCommentAction{ticketId, f.Get("comment")}, nil
}

func handleCreateComment(w http.ResponseWriter, r *http.Request) {
	repo := http_webui.LoadFromContext(r.Context(), &http_webui.ContextualRepoCache{}).(*http_webui.ContextualRepoCache).Repo
	bag := http_webui.LoadFromContext(r.Context(), &FlashMessageBag{}).(*FlashMessageBag)

	vars := mux.Vars(r)
	if err := r.ParseForm(); err != nil {
		ErrorIntoResponse(&malformedRequestError{prev: err}, w)
		return
	}

	ticketId := vars["ticketId"]
	action, err := submitCommentFromFormData(ticketId, r.Form)
	if err != nil {
		bag.AddMessage(NewError(err.Error()))
		ticketRedirect(ticketId, w, r)
		return
	}
	ticket, err := repo.ResolveBug(entity.Id(action.Ticket))
	if err != nil {
		ErrorIntoResponse(&invalidRequestError{msg: fmt.Sprintf("invalid ticket id: %s", action.Ticket)}, w)
		return
	}

	if err := addComment(ticket, action); err != nil {
		bag.AddMessage(NewError(fmt.Sprintf("Something went wrong: %s", err)))
	} else {
		bag.AddMessage(NewSuccess("Success"))
	}

	ticketRedirect(ticket.Id().String(), w, r)
}

func addComment(ticket *cache.BugCache, action *submitCommentAction) error {
	if _, err := ticket.AddComment(action.Comment); err != nil {
		return err
	}

	if err := ticket.CommitAsNeeded(); err != nil {
		return err
	}

	return nil
}

func ticketRedirect(ticketId string, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("/ticket/%s/", ticketId), http.StatusSeeOther)
}
