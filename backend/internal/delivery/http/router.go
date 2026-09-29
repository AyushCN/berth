package http

import (
	"github.com/AyushCN/berth/internal/config"
	"github.com/AyushCN/berth/internal/delivery/http/handler"
	"github.com/AyushCN/berth/internal/delivery/http/middleware"
	"github.com/gin-gonic/gin"
)

// NewRouter creates and configures the Gin router.
func NewRouter(cfg *config.Config, deps *Dependencies) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.FrontendURL))

	// Health check (no auth)
	r.GET("/health", handler.HealthCheck)

	// Preview Proxy (no auth, allows wildcards)
	r.Any("/p/:id/*path", deps.EnvironmentHandler.PreviewProxy)
	r.Any("/p/:id", deps.EnvironmentHandler.PreviewProxy)

	// API routes
	api := r.Group("/api")
	api.Use(middleware.RateLimit(cfg.RateLimitRequestsPerMinute))
	{
		// Auth
		api.GET("/auth/github", deps.AuthHandler.GithubLogin)
		api.GET("/auth/github/authorize", deps.AuthHandler.GithubAuthorize)
		api.GET("/auth/github/callback", deps.AuthHandler.GithubCallback)

		// Authenticated routes
		authenticated := api.Group("")
		authenticated.Use(middleware.Auth(cfg.JWTSecret))
		authenticated.Use(middleware.RateLimitUser(cfg.RateLimitAuthenticatedPerMinute))
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
			authenticated.GET("/projects/:id", deps.ProjectHandler.GetByID)
			authenticated.GET("/orgs/:id/projects", deps.ProjectHandler.ListForOrg)
			authenticated.GET("/projects/:id/sandboxes", deps.ProjectHandler.GetSandboxes)

			// Share Links
			authenticated.POST("/projects/:id/share-links", deps.ShareLinkHandler.CreateShareLink)
			authenticated.GET("/projects/:id/share-links", deps.ShareLinkHandler.GetShareLinks)
			authenticated.DELETE("/projects/:id/share-links/:linkId", deps.ShareLinkHandler.RevokeShareLink)

			// Join via share link (public endpoint, but requires auth)
			authenticated.POST("/join", deps.ShareLinkHandler.JoinViaShareLink)
			// Validate a share code without consuming a use. The /join/<code>
			// page calls this on mount and previously had no endpoint at all,
			// so every visitor got the "Link Invalid" screen.
			authenticated.GET("/share-links/validate", deps.ShareLinkHandler.ValidateShareLink)

			// Environments
			authenticated.GET("/environments", deps.EnvironmentHandler.ListEnvironments)
			authenticated.POST("/environments", deps.EnvironmentHandler.CreateEnvironment)
			authenticated.POST("/environments/:id/fork", deps.EnvironmentHandler.ForkEnvironment)
			authenticated.GET("/environments/:id", deps.EnvironmentHandler.GetEnvironment)
			authenticated.DELETE("/environments/:id", deps.EnvironmentHandler.DeleteEnvironment)
			authenticated.POST("/environments/:id/stop", deps.EnvironmentHandler.StopEnvironment)
			authenticated.POST("/environments/:id/restart", deps.EnvironmentHandler.RestartEnvironment)
			authenticated.POST("/environments/:id/start", deps.EnvironmentHandler.StartEnvironment)
			authenticated.POST("/environments/:id/exec", deps.EnvironmentHandler.ExecCommand)
			authenticated.GET("/environments/:id/logs", deps.EnvironmentHandler.GetLogs)

			// Files
			authenticated.GET("/environments/:id/files", deps.FileHandler.ListFiles)
			authenticated.GET("/environments/:id/files/content", deps.FileHandler.GetFileContent)
			authenticated.PUT("/environments/:id/files/content", deps.FileHandler.UpdateFileContent)
			authenticated.POST("/environments/:id/files/create", deps.FileHandler.CreateFile)
			authenticated.POST("/environments/:id/files/delete", deps.FileHandler.DeleteFile)

			// Git
			authenticated.GET("/environments/:id/git/status", deps.GitHandler.Status)
			authenticated.GET("/environments/:id/git/branches", deps.GitHandler.ListBranches)
			authenticated.POST("/environments/:id/git/branch", deps.GitHandler.CreateBranch)
			authenticated.POST("/environments/:id/git/checkout", deps.GitHandler.Checkout)
			authenticated.POST("/environments/:id/git/pull", deps.GitHandler.Pull)
			authenticated.POST("/environments/:id/git/commit", deps.GitHandler.Commit)
			authenticated.POST("/environments/:id/git/push", deps.GitHandler.Push)
			authenticated.GET("/environments/:id/git/log", deps.GitHandler.Log)

			// Change Requests
			authenticated.POST("/projects/:id/change-requests", deps.ChangeRequestHandler.CreateChangeRequest)
			authenticated.GET("/projects/:id/change-requests", deps.ChangeRequestHandler.ListChangeRequests)
			authenticated.GET("/change-requests/:id", deps.ChangeRequestHandler.GetChangeRequest)
			authenticated.PUT("/change-requests/:id", deps.ChangeRequestHandler.UpdateChangeRequest)
			authenticated.POST("/change-requests/:id/merge", deps.ChangeRequestHandler.MergeChangeRequest)
			authenticated.POST("/change-requests/:id/close", deps.ChangeRequestHandler.CloseChangeRequest)
			authenticated.GET("/change-requests/:id/diff", deps.ChangeRequestHandler.GetDiff)

			// Activity & Suspend/Resume
			authenticated.POST("/activity", deps.ActivityHandler.RecordActivity)
			authenticated.POST("/activity/session/start", deps.ActivityHandler.RecordSessionStart)
			authenticated.POST("/activity/session/end", deps.ActivityHandler.RecordSessionEnd)
			authenticated.POST("/environments/resume", deps.ActivityHandler.ResumeEnvironment)
			authenticated.GET("/environments/idle", deps.ActivityHandler.GetIdleEnvironments)
		}
	}

	// Development login mints a session for a fixed user signed with the real
	// JWT_SECRET. It must not exist in production, so it is not registered
	// there at all rather than relying on the handler to refuse.
	if cfg.Env != "production" {
		api.GET("/auth/dev-login", handler.DevLogin(cfg.JWTSecret, cfg.Env))
	}
	api.POST("/auth/logout", deps.AuthHandler.Logout)

	// Protected routes (auth via query param or cookie)
	ws := r.Group("/ws")
	ws.Use(middleware.WSAuth(cfg.JWTSecret))
	{
		ws.GET("/environments/:id", deps.WSHandler.HandleSandboxWS)
		ws.GET("/sandbox/:id", deps.WSHandler.HandleSandboxWS)
		ws.GET("/sandboxes/:id", deps.WSHandler.HandleSandboxWS)
	}

	return r
}

// Dependencies holds all handler dependencies.
type Dependencies struct {
	AuthHandler          *handler.AuthHandler
	EnvironmentHandler   *handler.EnvironmentHandler
	FileHandler          *handler.FileHandler
	WSHandler            *handler.WSHandler
	GitHandler           *handler.GitHandler
	OrgHandler           *handler.OrganizationHandler
	ProjectHandler       *handler.ProjectHandler
	ShareLinkHandler     *handler.ShareLinkHandler
	ChangeRequestHandler *handler.ChangeRequestHandler
	ActivityHandler      *handler.ActivityHandler
}
