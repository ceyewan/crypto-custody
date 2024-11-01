package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Auth is a middleware function for authentication
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Here you would typically validate the token
		// For example, you could call a function like validateToken(token)

		c.Next()
	}
}

// RequireRole is a middleware function for role-based authorization
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Here you would typically extract the user's roles from the context or token
		// For example, you could call a function like getUserRoles(c)

		userRoles := []string{"system_admin"} // This is just a placeholder

		hasRole := false
		for _, r := range userRoles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RequireRoleOrPermission(role, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Here you would typically extract the user's roles and permissions from the context or token
		// For example, you could call a function like getUserRolesAndPermissions(c)

		userRoles := []string{"system_admin"}     // This is just a placeholder
		userPermissions := []string{"case_write"} // This is just a placeholder

		hasRole := false
		for _, r := range userRoles {
			if r == role {
				hasRole = true
				break
			}
		}

		hasPermission := false
		for _, p := range userPermissions {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasRole && !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			c.Abort()
			return
		}

		c.Next()
	}
}
