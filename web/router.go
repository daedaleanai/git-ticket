package web

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/daedaleanai/git-ticket/cache"
)

//go:embed assets
var assets embed.FS

//go:embed templates
var templates embed.FS

type WebEnvFactory struct {
	MakeEnv    func() (*cache.RepoCache, error)
	ReleaseEnv func(*cache.RepoCache)
}

// TODO: This function is a reason why server is likely to melt under load:
// it locks the repository and recreates the state at every request.
// Otherwise you could not run the server in parallel with any other `git ticket` operation (e.g. pull).
// This is good enough for `localhost` but shall be changed when making it a proper frontend.
func lockAndExecute(envFactory *WebEnvFactory, handler func(*gin.Context, *cache.RepoCache)) func(*gin.Context) {
	return func(c *gin.Context) {
		env, err := envFactory.MakeEnv()
		if err != nil {
			log.Fatalf("Failed to allocate environment: %s", err)
		}
		defer envFactory.ReleaseEnv(env)

		handler(c, env)
	}
}

func MakeRouter(envFactory *WebEnvFactory) *gin.Engine {
	assetsFs, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Fatalf("Assets must have been embedded into the binary: %s", err)
	}

	parsedTemplates, err := template.ParseFS(templates, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse embedded templates: %s", err)
	}

	r := gin.Default()
	r.SetHTMLTemplate(parsedTemplates)

	// TODO: cors.Default() is free dinner for everyone. Shall be DefaultConfig() of sorts.
	r.Use(cors.Default())

	r.StaticFS("/assets", http.FS(assetsFs))
	r.GET("/", lockAndExecute(envFactory, handleTicketsList))
	r.GET("/tickets", lockAndExecute(envFactory, handleTicketsList))
	r.GET("/ticket/:id", lockAndExecute(envFactory, handleTicket))
	r.GET("/personas", lockAndExecute(envFactory, handlePersonasList))
	r.GET("/persona/:id", lockAndExecute(envFactory, handlePersona))
	return r
}
