package middleware

import "github.com/gin-gonic/gin"

// extractContextValue extracts a value from gin context with type assertion
func extractContextValue(c *gin.Context, key string) (string, bool) {
	value, exists := c.Get(key)
	if !exists {
		return "", false
	}
	strValue, ok := value.(string)
	return strValue, ok
}

// extractBoolFromContext extracts a boolean value from gin context
func extractBoolFromContext(c *gin.Context, key string) (bool, bool) {
	value, exists := c.Get(key)
	if !exists {
		return false, false
	}
	boolValue, ok := value.(bool)
	return boolValue, ok
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
