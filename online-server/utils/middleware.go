package utils

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")
		if authorization == "" {
			c.JSON(200, "NeedToken")
			c.Abort()
			return
		}

		parts := strings.Split(authorization, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(200, "TokenFormatErr")
			c.Abort()
			return
		}

		claims, err := ValidateJWT(parts[1])
		if err != nil {
			c.JSON(200, "TokenErr")
			c.Abort()
			return
		}

		if time.Now().Unix() > claims.ExpiresAt.Unix() {
			c.JSON(200, "TokenExpiration")
			c.Abort()
			return
		}
		if claims.RoleID != 1 {
			c.JSON(403, "Forbidden")
			c.Abort()
			return
		}

		c.Set("UserId", claims.UserID)
		c.Set("UserName", claims.Username)
		c.Set("Role", claims.RoleID)

		c.Next()
	}
}
