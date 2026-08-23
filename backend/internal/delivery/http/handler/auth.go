package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/AyushCN/berth/internal/usecase"
)

// AuthHandler handles authentication HTTP requests.
type AuthHandler struct {
	authUC *usecase.AuthUsecase
}

func NewAuthHandler(uc *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUC: uc}
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

	// Store verifier in cookie (secure, httpOnly)
	c.SetCookie("pkce_verifier", verifier, 600, "/", "", true, true)
	c.SetCookie("oauth_state", state, 600, "/", "", true, true)

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

	token, user, err := h.authUC.ProcessCallback(c.Request.Context(), code, verifier)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Set JWT cookie
	c.SetCookie("berth_token", token, 86400, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
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
