package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// cookieSecure reports whether the request arrived over TLS.
//
// Traefik terminates TLS and forwards the original scheme, so the
// X-Forwarded-Proto header has to be trusted when present.
//
// Deriving this per request matters: a cookie marked Secure is *silently
// dropped* by browsers when the page was loaded over plain http, so hardcoding
// Secure=true breaks every login on a plain-http deployment (which is the
// documented local setup) with no error anywhere to explain why.
func cookieSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return c.GetHeader("X-Forwarded-Proto") == "https"
}

// setAuthCookie writes a session cookie with attributes derived from the
// request rather than hardcoded.
//
// SameSite is left unset, which Go's SetCookie omits entirely; browsers then
// apply their default of Lax. That is what the OAuth callback needs: the
// state and verifier cookies are read on a top-level navigation back from
// github.com, which Lax permits. Note that SameSite=None would *require*
// Secure, so the two cannot be combined on a plain-http deployment.
func setAuthCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", cookieSecure(c), true)
}

// clearAuthCookie expires a cookie using the same attributes it was set with,
// otherwise the browser keeps the original.
func clearAuthCookie(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, "", -1, "/", "", cookieSecure(c), true)
}

const (
	berthTokenCookie   = "berth_token"
	oauthStateCookie   = "oauth_state"
	pkceVerifierCookie = "pkce_verifier"
	sessionMaxAge      = 24 * 60 * 60
	oauthFlowMaxAge    = 600
)
