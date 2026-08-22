package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "time"
)

func DevLogin(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := jwt.MapClaims{
            "userId": "00000000-0000-0000-0000-000000000001",
            "exp":    time.Now().Add(24 * time.Hour).Unix(),
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        tokenString, _ := token.SignedString([]byte(jwtSecret))
        c.SetCookie("berth_token", tokenString, 86400, "/", "", false, true)
        c.JSON(http.StatusOK, gin.H{"token": tokenString})
    }
}
