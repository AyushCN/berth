package handler

import (
	"net/http"

	"github.com/AyushCN/berth/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	authUC      *usecase.AuthUsecase
	frontendURL string
}

func NewAuthHandler(uc *usecase.AuthUsecase, frontendURL string) *AuthHandler {
	return &AuthHandler{
		authUC:      uc,
		frontendURL: frontendURL,
	}
}

// GithubLogin initiates the OAuth flow.
func (h *AuthHandler) GithubLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "use /api/auth/github/authorize for redirect",
	})
}

// GithubAuthorize redirects to GitHub OAuth.
func (h *AuthHandler) GithubAuthorize(c *gin.Context) {
	url, state, verifier, err := h.authUC.GenerateAuthorizeURL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate authorize url"})
		return
	}

	// Store the PKCE verifier and state. Secure is derived from the request:
	// hardcoding it broke login on any plain-http deployment, because browsers
	// silently discard a Secure cookie served over http.
	setAuthCookie(c, pkceVerifierCookie, verifier, oauthFlowMaxAge)
	setAuthCookie(c, oauthStateCookie, state, oauthFlowMaxAge)

	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GithubCallback handles the OAuth callback.
func (h *AuthHandler) GithubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	verifier, err := c.Cookie("pkce_verifier")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing pkce verifier"})
		return
	}

	expectedState, err := c.Cookie("oauth_state")
	if err != nil || state != expectedState {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	token, _, err := h.authUC.ProcessCallback(c.Request.Context(), code, verifier)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	setAuthCookie(c, berthTokenCookie, token, sessionMaxAge)

	// Redirect to frontend dashboard
	c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/dashboard")
}

// GetMe returns the current authenticated user.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userIDStr, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	user, err := h.authUC.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Logout expires the browser's HttpOnly session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	clearAuthCookie(c, berthTokenCookie)
	c.Status(http.StatusNoContent)
}
