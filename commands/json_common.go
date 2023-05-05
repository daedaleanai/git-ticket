package commands

import (
	"time"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/identity"
	"github.com/daedaleanai/git-ticket/util/lamport"
)

type JSONIdentity struct {
	Id      string `json:"id"`
	HumanId string `json:"human_id"`
	Name    string `json:"name"`
	Login   string `json:"login"`
}

func NewJSONIdentity(i identity.Interface) JSONIdentity {
	return JSONIdentity{
		Id:      i.Id().String(),
		HumanId: i.Id().Human(),
		Name:    i.Name(),
		Login:   i.Login(),
	}
}

func NewJSONIdentityFromExcerpt(excerpt *cache.IdentityExcerpt) JSONIdentity {
	return JSONIdentity{
		Id:      excerpt.Id.String(),
		HumanId: excerpt.Id.Human(),
		Name:    excerpt.Name,
		Login:   excerpt.Login,
	}
}

func NewJSONIdentityFromLegacyExcerpt(excerpt *cache.LegacyAuthorExcerpt) JSONIdentity {
	return JSONIdentity{
		Name:  excerpt.Name,
		Login: excerpt.Login,
	}
}

type JSONCcbInfo struct {
	User   JSONIdentity
	Status string
	State  string
}

type JSONTime struct {
	Timestamp int64        `json:"timestamp"`
	Time      time.Time    `json:"time"`
	Lamport   lamport.Time `json:"lamport,omitempty"`
}

func NewJSONTime(t time.Time, l lamport.Time) JSONTime {
	return JSONTime{
		Timestamp: t.Unix(),
		Time:      t,
		Lamport:   l,
	}
}
