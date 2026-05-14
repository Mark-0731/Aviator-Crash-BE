package utils

import (
	"errors"
	"time"

	"aviator-backend/config"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a JWT access token
func GenerateAccessToken(userID primitive.ObjectID, username string, isAdmin bool) (string, error) {
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWTAccessExpiryMinutes) * time.Minute)

	claims := &Claims{
		UserID:   userID.Hex(),
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// GenerateRefreshToken generates a JWT refresh token
func GenerateRefreshToken(userID primitive.ObjectID) (string, error) {
	expirationTime := time.Now().Add(time.Duration(config.AppConfig.JWTRefreshExpiryDays) * 24 * time.Hour)

	claims := &Claims{
		UserID: userID.Hex(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ExtractUserIDFromToken extracts user ID from token string
func ExtractUserIDFromToken(tokenString string) (primitive.ObjectID, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return primitive.NilObjectID, err
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return primitive.NilObjectID, errors.New("invalid user ID in token")
	}

	return userID, nil
}
