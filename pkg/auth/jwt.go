package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/FACorreiaa/go-templui/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your-secret-key-change-in-production")

type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (a *AuthService) GenerateToken(userID, email, name string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		logger.Log.Error("Failed to generate JWT token", 
			zap.Error(err),
		)
		return "", err
	}

	return tokenString, nil
}

func (a *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (a *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Log.Error("Failed to hash password",
			zap.Error(err),
		)
		return "", err
	}
	return string(bytes), nil
}

func (a *AuthService) CheckPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Check for cookie as fallback
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				logger.Log.Info("No auth token provided",
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			logger.Log.Warn("Invalid authorization header format")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		auth := NewAuthService()
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			logger.Log.Warn("Invalid JWT token",
				zap.Error(err),
			)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Add user info to request context
		r = r.WithContext(context.WithValue(r.Context(), "user_id", claims.UserID))
		r = r.WithContext(context.WithValue(r.Context(), "user_email", claims.Email))
		r = r.WithContext(context.WithValue(r.Context(), "user_name", claims.Name))

		next.ServeHTTP(w, r)
	})
}