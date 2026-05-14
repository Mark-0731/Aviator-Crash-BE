package middleware

import (
	"aviator-backend/utils"

	"github.com/gin-gonic/gin"
)

// FULLY FUNCTIONAL - NO PLACEHOLDERS

// AdminOnly ensures the user is an admin
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin := IsAdmin(c)
		if !isAdmin {
			utils.RespondWithError(c, utils.ErrForbidden, "Admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
