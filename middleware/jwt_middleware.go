package middleware

import (
	"strings"

	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

// JWTAuth validates JWT token and adds user info to context
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.RespondWithError(c, utils.ErrUnauthorized, "Authorization header required")
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.RespondWithError(c, utils.ErrUnauthorized, "Invalid authorization format")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.RespondWithError(c, utils.ErrUnauthorized, "Invalid or expired token")
			c.Abort()
			return
		}

		// Add user info to context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) string {
	userID, _ := extractContextValue(c, "user_id")
	return userID
}

// GetUsername extracts username from context
func GetUsername(c *gin.Context) string {
	username, _ := extractContextValue(c, "username")
	return username
}

// IsAdmin checks if user is admin
func IsAdmin(c *gin.Context) bool {
	isAdmin, _ := extractBoolFromContext(c, "is_admin")
	return isAdmin
}
