package controllers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// parsePagination parses and validates pagination parameters
func parsePagination(c *gin.Context, defaultLimit int64) (page, limit int64) {
	page, _ = strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ = strconv.ParseInt(c.DefaultQuery("limit", strconv.FormatInt(defaultLimit, 10)), 10, 64)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = defaultLimit
	}

	return page, limit
}

// parseObjectID parses and validates MongoDB ObjectID from string
func parseObjectID(idStr string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(idStr)
}

// paginationResponse creates a standard pagination response
func paginationResponse(data any, total, page, limit int64) gin.H {
	return gin.H{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	}
}
