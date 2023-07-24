package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/daedaleanai/git-ticket/cache"
	"github.com/daedaleanai/git-ticket/entity"
)

func _personasListPage(c *gin.Context, excerpts []*cache.IdentityExcerpt) {
	c.HTML(http.StatusOK, "personas.html", gin.H{
		"title":    "Personas",
		"excerpts": excerpts,
	})
}

func _personaPage(c *gin.Context, persona *cache.IdentityExcerpt) {
	c.HTML(http.StatusOK, "persona.html", gin.H{
		"title":   persona.DisplayName(),
		"persona": persona,
	})
}

func handlePersonasList(c *gin.Context, backend *cache.RepoCache) {
	personaExcerpts := []*cache.IdentityExcerpt{}
	for _, id := range backend.AllIdentityIds() {
		p, err := backend.ResolveIdentityExcerpt(id)
		if p == nil || err != nil {
			continue
		}
		personaExcerpts = append(personaExcerpts, p)
	}
	_personasListPage(c, personaExcerpts)
}

func handlePersona(c *gin.Context, backend *cache.RepoCache) {
	if p, err := backend.ResolveIdentityExcerpt(entity.Id(c.Param("id"))); err == nil {
		_personaPage(c, p)
	}
}
