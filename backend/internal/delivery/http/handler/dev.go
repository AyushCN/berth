package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// DevUserID is the fixed identity minted by the development login route.
const DevUserID = "00000000-0000-0000-0000-000000000001"

// DevLogin mints a session for the seeded development user.
//
// This route must never be registered in production. It hands out a valid
// HS256 token signed with the real JWT_SECRET to anyone who asks, so registering
// it outside development is an unauthenticated authentication bypass. It was
// previously registered unconditionally, with `env` only used to pick the
// cookie's Secure flag.
func DevLogin(jwtSecret string, env string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Defence in depth: the route is not registered in production, but if
		// it is ever wired up again by accident, refuse rather than mint.
		if env == "production" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		claims := jwt.MapClaims{
			"userId": DevUserID,
			"exp":    time.Now().Add(24 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to sign token"})
			return
		}
		// Never Secure over plain http: the browser silently drops such
		// cookies, which is why dev login has to keep this false. The OAuth
		// callback hardcodes true, which is a separate inconsistency.
		c.SetCookie("berth_token", tokenString, 86400, "/", "", false, true)
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}
