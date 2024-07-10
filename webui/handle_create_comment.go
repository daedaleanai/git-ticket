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

func handleCreateComment(repo *cache.RepoCache, w http.ResponseWriter, r *http.Request) error {
	bag := http_webui.LoadFromContext(r.Context(), &FlashMessageBag{}).(*FlashMessageBag)

	vars := mux.Vars(r)
	if err := r.ParseForm(); err != nil {
		return &malformedRequestError{prev: err}
	}

	ticketId := vars["ticketId"]
	action, err := submitCommentFromFormData(ticketId, r.Form)
	if err != nil {
		bag.Add(NewError(err.Error()))
		ticketRedirect(ticketId, w, r)
		return nil
	}

	ticket, err := repo.ResolveBug(entity.Id(action.Ticket))
	if err != nil {
		return &invalidRequestError{msg: fmt.Sprintf("invalid ticket id: %s", action.Ticket)}
	}

	_, err = ticket.AddComment(action.Comment)
	if err != nil {
		bag.Add(NewError(fmt.Sprintf("Something went wrong: %s", err)))
	}

	if err := ticket.CommitAsNeeded(); err != nil {
		bag.Add(NewError(fmt.Sprintf("Something went wrong: %s", err)))
	}

	if err == nil {
		bag.Add(NewSuccess("Success"))
	}

	ticketRedirect(ticket.Id().String(), w, r)
	return nil
}

func ticketRedirect(ticketId string, w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, fmt.Sprintf("/ticket/%s/", ticketId), http.StatusSeeOther)
}
