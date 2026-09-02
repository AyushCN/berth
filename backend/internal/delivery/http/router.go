package http

import (
	"github.com/gin-gonic/gin"
	"github.com/AyushCN/berth/internal/config"
	"github.com/AyushCN/berth/internal/delivery/http/handler"
	"github.com/AyushCN/berth/internal/delivery/http/middleware"
)

// NewRouter creates and configures the Gin router.
func NewRouter(cfg *config.Config, deps *Dependencies) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Health check (no auth)
	r.GET("/health", handler.HealthCheck)

	// API routes
	api := r.Group("/api")
	// api.Use(middleware.RateLimit())
	{
		// Auth
		api.GET("/auth/github", deps.AuthHandler.GithubLogin)
		api.GET("/auth/github/authorize", deps.AuthHandler.GithubAuthorize)
		api.GET("/auth/github/callback", deps.AuthHandler.GithubCallback)

		// Authenticated routes
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(cfg.JWTSecret))
		// authenticated.Use(middleware.RateLimitUser())
		{
			authenticated.GET("/user/me", deps.AuthHandler.GetMe)

			// Organizations
			authenticated.POST("/orgs", deps.OrgHandler.Create)
			authenticated.GET("/orgs", deps.OrgHandler.List)
			authenticated.POST("/orgs/:id/members", deps.OrgHandler.AddMember)
			authenticated.GET("/orgs/:id/members", deps.OrgHandler.ListMembers)

			// Projects
			authenticated.POST("/projects", deps.ProjectHandler.Create)
			authenticated.GET("/projects", deps.ProjectHandler.ListForUser)
			authenticated.GET("/orgs/:id/projects", deps.ProjectHandler.ListForOrg)
			authenticated.GET("/projects/:id/sandboxes", deps.ProjectHandler.GetSandboxes)

			// Environments
			authenticated.GET("/environments", deps.SandboxHandler.ListEnvironments)
			authenticated.POST("/environments", deps.SandboxHandler.CreateEnvironment)
			authenticated.POST("/environments/:id/fork", deps.SandboxHandler.ForkEnvironment)
			authenticated.GET("/environments/:id", deps.SandboxHandler.GetEnvironment)
			authenticated.DELETE("/environments/:id", deps.SandboxHandler.DeleteEnvironment)
			authenticated.POST("/environments/:id/exec", deps.SandboxHandler.ExecCommand)
			authenticated.GET("/environments/:id/logs", deps.SandboxHandler.GetLogs)

			// Files
			authenticated.GET("/environments/:id/files", deps.FileHandler.ListFiles)
			authenticated.GET("/environments/:id/files/content", deps.FileHandler.GetFileContent)
			authenticated.PUT("/environments/:id/files/content", deps.FileHandler.UpdateFileContent)

			// Git
			authenticated.GET("/environments/:id/git/status", deps.GitHandler.Status)
			authenticated.GET("/environments/:id/git/branches", deps.GitHandler.ListBranches)
			authenticated.POST("/environments/:id/git/branch", deps.GitHandler.CreateBranch)
			authenticated.POST("/environments/:id/git/checkout", deps.GitHandler.Checkout)
			authenticated.POST("/environments/:id/git/pull", deps.GitHandler.Pull)
			authenticated.POST("/environments/:id/git/commit", deps.GitHandler.Commit)
			authenticated.POST("/environments/:id/git/push", deps.GitHandler.Push)
			authenticated.GET("/environments/:id/git/log", deps.GitHandler.Log)
		}
	}

	api.GET("/auth/dev-login", handler.DevLogin(cfg.JWTSecret, cfg.Env))

	// Protected routes (auth via query param or cookie)
	ws := r.Group("/ws")
	ws.Use(middleware.WSAuth(cfg.JWTSecret))
	{
		ws.GET("/sandbox/:id", deps.WSHandler.HandleSandboxWS)
		ws.GET("/file/:id", deps.WSHandler.HandleFileSyncWS)
	}

	return r
}

// Dependencies holds all handler dependencies.
type Dependencies struct {
	AuthHandler    *handler.AuthHandler
	SandboxHandler *handler.SandboxHandler
	FileHandler    *handler.FileHandler
	WSHandler      *handler.WSHandler
	GitHandler     *handler.GitHandler
	OrgHandler     *handler.OrganizationHandler
	ProjectHandler *handler.ProjectHandler
}
