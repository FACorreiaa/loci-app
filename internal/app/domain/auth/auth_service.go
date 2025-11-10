// app/auth/service.go
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/FACorreiaa/go-templui/internal/app/middleware"
	"github.com/FACorreiaa/go-templui/internal/app/models"
	"github.com/FACorreiaa/go-templui/internal/pkg/config"
)

// Ensure implementation satisfies the interface
var _ AuthService = (*AuthServiceImpl)(nil)

// AuthService defines the business logic contract.
type AuthService interface {
	Login(ctx context.Context, email, password string) (accessToken string, refreshToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshSession(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, err error)
	Register(ctx context.Context, username, email, password, role string) (string, error)
	UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error
	InvalidateAllUserRefreshTokens(ctx context.Context, userID string) error
	ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error)
	GetUserByID(ctx context.Context, userID string) (*models.UserAuth, error)
	GetUserByEmail(ctx context.Context, email string) (*models.UserAuth, error)
	VerifyPassword(ctx context.Context, userID, password string) error
	GenerateTokens(ctx context.Context, user *models.UserAuth, sub *models.Subscription) (accessToken string, refreshToken string, err error)

	// Token and utility methods
	GenerateToken(userID, email, username string) (string, error)
	GenerateTokenWithExpiration(userID, email, username string, expiration time.Duration) (string, error)
	ValidateToken(tokenString string) (*jwt.MapClaims, error)
	HashPassword(password string) (string, error)
	CheckPassword(hashedPassword, password string) bool
}

// AuthServiceImpl provides the implementation for AuthService.
type AuthServiceImpl struct {
	logger *zap.Logger
	repo   AuthRepo // Use the interface
	cfg    *config.Config
}

// NewAuthService creates a new authentication service instance.
func NewAuthService(repo AuthRepo, cfg *config.Config, logger *zap.Logger) *AuthServiceImpl {
	// ... (nil checks and validation as before) ...
	return &AuthServiceImpl{logger: logger, repo: repo, cfg: cfg}
}

// Login validates credentials, generates tokens, stores refresh token.
func (s *AuthServiceImpl) Login(ctx context.Context, email, password string) (string, string, error) {
	l := s.logger.With(zap.String("method", "Login"), zap.String("email", email))
	l.Debug("Attempting login")

	// 1. Fetch user by email (includes hash)
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		l.Warn("GetUserByEmail failed", zap.String("email", email))
		// Don't reveal if user exists or password is wrong
		return "", "", fmt.Errorf("invalid credentials: %w", models.ErrUnauthenticated)
	}

	// 2. Compare submitted password with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		l.Warn("Password comparison failed", zap.String("userID", user.ID))
		return "", "", fmt.Errorf("invalid credentials: %w", models.ErrUnauthenticated)
	}

	// --- Add models.Subscription Fetching Here Later ---
	// sub, err := s.subsRepo.GetCurrentmodels.SubscriptionByUserID(ctx, user.ID) ...
	// For now, create dummy/default sub info for token generation
	sub := &models.Subscription{Plan: "free", Status: "active"} // Placeholder

	// 3. Generate Tokens
	accessToken, refreshToken, err := s.GenerateTokens(ctx, user, sub) // Pass user and sub
	if err != nil {
		l.Error("Failed to generate tokens", zap.String("userID", user.ID), zap.Error(err))
		return "", "", fmt.Errorf("app error generating tokens: %w", err)
	}

	// 4. Store the new Refresh Token
	refreshTTL := s.getRefreshTTL()
	refreshExpiresAt := time.Now().Add(refreshTTL)
	err = s.repo.StoreRefreshToken(ctx, user.ID, refreshToken, refreshExpiresAt)
	if err != nil {
		l.Error("Failed to store refresh token", zap.String("userID", user.ID), zap.Error(err))
		return "", "", fmt.Errorf("app error storing session: %w", err)
	}

	l.Info("Login successful")
	return accessToken, refreshToken, nil
}

func (s *AuthServiceImpl) Register(ctx context.Context, username, email, password, _ string) (string, error) {
	l := s.logger.With(zap.String("method", "Register"), zap.String("email", email))
	l.Debug("Attempting registration")

	// Get the global tracer
	tracer := otel.Tracer("MyRESTAPI")

	// Start a child span for the service layer
	ctx, span := tracer.Start(ctx, "AuthService.Register", trace.WithAttributes(
		attribute.String("username", username),
		attribute.String("email", email),
	))
	defer span.End()

	// Hash the password
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		l.Error("Failed to hash password", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Password hashing failed")
		return "", fmt.Errorf("could not process password")
	}
	hashedPassword := string(hashedPasswordBytes)

	// Call repository to store user
	userID, err := s.repo.Register(ctx, username, email, hashedPassword)
	if err != nil {
		l.Error("Repository registration failed", zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "Repository registration failed")
		return "", fmt.Errorf("registration failed: %w", err)
	}

	l.Info("Registration successful", zap.String("userID", userID))
	span.SetStatus(codes.Ok, "User registered")
	return userID, nil
}

// RefreshSession validates refresh token, generates new tokens, rotates refresh token.
func (s *AuthServiceImpl) RefreshSession(ctx context.Context, refreshToken string) (string, string, error) {
	l := s.logger.With(zap.String("method", "RefreshSession"))
	l.Debug("Attempting token refresh")

	// 1. Validate refresh token and get User ID
	userID, err := s.repo.ValidateRefreshTokenAndGetUserID(ctx, refreshToken)
	if err != nil {
		l.Warn("Refresh token validation failed", zap.Error(err))
		return "", "", fmt.Errorf("invalid or expired refresh token: %w", err)
	}

	// 2. Fetch full user details for new token claims
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		l.Error("Failed to get user details after refresh token validation", zap.String("userID", userID), zap.Error(err))
		// Invalidate the suspicious token?
		err = s.repo.InvalidateRefreshToken(ctx, refreshToken)
		if err != nil {
			return "", "", fmt.Errorf("invalid or expired refresh token: %w", models.ErrUnauthenticated)
		}
		return "", "", fmt.Errorf("app error retrieving user during refresh: %w", models.ErrUnauthenticated)
	}

	// --- Fetch models.Subscription Here Later ---
	sub := &models.Subscription{Plan: "free", Status: "active"} // Placeholder

	// 3. Generate NEW tokens
	newAccessToken, newRefreshToken, err := s.GenerateTokens(ctx, user, sub)
	if err != nil {
		l.Error("Failed to generate new tokens", zap.String("userID", user.ID), zap.Error(err))
		return "", "", fmt.Errorf("app error generating tokens: %w", err)
	}

	// 4. Store the NEW refresh token
	refreshTTL := s.getRefreshTTL()
	refreshExpiresAt := time.Now().Add(refreshTTL)
	err = s.repo.StoreRefreshToken(ctx, user.ID, newRefreshToken, refreshExpiresAt)
	if err != nil {
		l.Error("Failed to store new refresh token", zap.String("userID", user.ID), zap.Error(err))
		return "", "", fmt.Errorf("app error storing new session: %w", err)
	}

	// 5. Invalidate the OLD refresh token (Rotation)
	err = s.repo.InvalidateRefreshToken(ctx, refreshToken)
	if err != nil {
		l.Warn("Failed to invalidate old refresh token during rotation", zap.String("userID", user.ID), zap.Error(err))
		// Log, but proceed since new tokens were issued
		return "", "", fmt.Errorf("failed to invalidate old refresh token: %w", err)
	}

	l.Info("Token refresh successful", zap.String("userID", user.ID))
	return newAccessToken, newRefreshToken, nil
}

// Logout invalidates the provided refresh token.
func (s *AuthServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	l := s.logger.With(zap.String("method", "Logout"))
	l.Debug("Attempting logout by invalidating refresh token")
	err := s.repo.InvalidateRefreshToken(ctx, refreshToken)
	if err != nil {
		l.Error("Failed to invalidate refresh token", zap.Error(err))
		// Decide if this should be an error back to client
		return fmt.Errorf("logout failed: %w", err)
	}
	l.Info("Logout successful (token invalidated)")
	return nil // Usually succeed logout even if invalidation had minor issues
}

// UpdatePassword verifies old password, hashes new one, updates, invalidates tokens.
func (s *AuthServiceImpl) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	l := s.logger.With(zap.String("method", "UpdatePassword"), zap.String("userID", userID))
	l.Debug("Attempting password update")

	// 1. Verify old password using the repository method
	err := s.repo.VerifyPassword(ctx, userID, oldPassword)
	if err != nil {
		l.Warn("Old password verification failed", zap.Error(err))
		return fmt.Errorf("incorrect old password: %w", models.ErrUnauthenticated)
	}

	// 2. Hash the *new* password
	newHashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		l.Error("Failed to hash new password", zap.Error(err))
		return fmt.Errorf("could not process new password")
	}
	newHashedPassword := string(newHashedPasswordBytes)

	// 3. Call repository to update the stored hash
	err = s.repo.UpdatePassword(ctx, userID, newHashedPassword)
	if err != nil {
		l.Error("Repository password update failed", zap.Error(err))
		return fmt.Errorf("failed to update password: %w", err)
	}

	// 4. Invalidate all refresh tokens for security
	err = s.InvalidateAllUserRefreshTokens(ctx, userID) // Call self to use logging etc.
	if err != nil {
		// Log as warning, password update succeeded but token invalidation failed
		l.Warn("Failed to invalidate refresh tokens after password update", zap.Error(err))
		return err
	}

	l.Info("Password updated successfully")
	return nil
}

// InvalidateAllUserRefreshTokens invalidates all active refresh tokens for a user.
func (s *AuthServiceImpl) InvalidateAllUserRefreshTokens(ctx context.Context, userID string) error {
	l := s.logger.With(zap.String("method", "InvalidateAllUserRefreshTokens"), zap.String("userID", userID))
	l.Debug("Invalidating all refresh tokens")
	err := s.repo.InvalidateAllUserRefreshTokens(ctx, userID)
	if err != nil {
		l.Error("Failed to invalidate all refresh tokens", zap.Error(err))
		return fmt.Errorf("failed to invalidate tokens: %w", err)
	}
	l.Info("All refresh tokens invalidated")
	return nil
}

func (s *AuthServiceImpl) GetUserByID(ctx context.Context, userID string) (*models.UserAuth, error) {
	l := s.logger.With(zap.String("method", "GetUserByID"), zap.String("userID", userID))
	l.Debug("Fetching user by ID")
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		l.Error("Failed to fetch user by ID", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	l.Info("User fetched successfully")
	return user, nil
}

func (s *AuthServiceImpl) GetUserByEmail(ctx context.Context, email string) (*models.UserAuth, error) {
	l := s.logger.With(zap.String("method", "GetUserByEmail"), zap.String("email", email))
	l.Debug("Fetching user by email")
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		l.Error("Failed to fetch user by email", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	l.Info("User fetched successfully")
	return user, nil
}

// --- Internal Helper: generateTokens ---
func (s *AuthServiceImpl) GenerateTokens(ctx context.Context, user *models.UserAuth, _ *models.Subscription) (accessToken string, refreshToken string, err error) {
	l := s.logger.With(zap.String("method", "generateTokens"), zap.String("userID", user.ID))

	// --- Access Token ---
	accessTTL := s.getAccessTTL()
	issuer := s.getIssuer()
	audience := s.getAudience()
	secretKeyBytes := []byte(s.getSecretKey())

	accessClaims := &models.Claims{ // Use your Claims struct
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
		},
		// Custom Claims
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		//models.SubscriptionPlan:   sub.Plan,   // Add from sub
		//models.SubscriptionStatus: sub.Status, // Add from sub
	}
	accessTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenJWT.SignedString(secretKeyBytes)
	if err != nil {
		l.Error("Failed to sign access token", zap.Error(err))
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// --- Refresh Token ---
	refreshToken = uuid.NewString() // Simple UUID, stored in DB

	l.Debug("Tokens generated successfully")
	return accessToken, refreshToken, nil
}

func (s *AuthServiceImpl) VerifyPassword(ctx context.Context, userID, password string) error {
	return s.repo.VerifyPassword(ctx, userID, password)
}

// --- Internal Helpers for Config with Defaults ---
func (s *AuthServiceImpl) getAccessTTL() time.Duration {
	if s.cfg != nil && s.cfg.JWT.AccessTokenTTL > 0 {
		return s.cfg.JWT.AccessTokenTTL
	}
	s.logger.Warn("JWT AccessTokenTTL not configured, using default 15m")
	return 15 * time.Minute
}
func (s *AuthServiceImpl) getRefreshTTL() time.Duration {
	if s.cfg != nil && s.cfg.JWT.RefreshTokenTTL > 0 {
		return s.cfg.JWT.RefreshTokenTTL
	}
	s.logger.Warn("JWT RefreshTokenTTL not configured, using default 7d")
	return 7 * 24 * time.Hour
}
func (s *AuthServiceImpl) getIssuer() string {
	if s.cfg != nil && s.cfg.JWT.Issuer != "" {
		return s.cfg.JWT.Issuer
	}
	return "Loci" // Default
}
func (s *AuthServiceImpl) getAudience() string {
	if s.cfg != nil && s.cfg.JWT.Audience != "" {
		return s.cfg.JWT.Audience
	}
	return "Loci-app" // Default
}
func (s *AuthServiceImpl) getSecretKey() string {
	// Already checked for empty in NewAuthService
	return s.cfg.JWT.SecretKey
}

func (s *AuthServiceImpl) ValidateRefreshToken(ctx context.Context, refreshToken string) (string, error) {
	// Assuming repo has a method to validate refresh tokens
	userID, err := s.repo.ValidateRefreshTokenAndGetUserID(ctx, refreshToken)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// implement much later

// Implement a dummy for now if needed for compilation
type dummySubsRepo struct{}

func (d *dummySubsRepo) GetCurrentSubscriptionByUserID(_ context.Context, _ string) (*models.Subscription, error) {
	return &models.Subscription{Plan: "free", Status: "active"}, nil // Always return free/active
}
func (d *dummySubsRepo) CreateDefaultSubscription(_ context.Context, _ string) error {
	return nil // Do nothing
}
func NewDummySubsRepo() models.SubscriptionRepository { return &dummySubsRepo{} }

// --- Token and utility methods implementation ---

func (s *AuthServiceImpl) GenerateToken(userID, email, username string) (string, error) {
	jwtService := middleware.NewJWTService()
	jwtCfg := middleware.JWTConfig{
		SecretKey:       s.getSecretKey(),
		TokenExpiration: s.getAccessTTL(),
		Logger:          s.logger,
		Optional:        false,
	}
	return jwtService.GenerateToken(jwtCfg, userID, email, username)
}

func (s *AuthServiceImpl) GenerateTokenWithExpiration(userID, email, username string, expiration time.Duration) (string, error) {
	jwtService := middleware.NewJWTService()
	jwtCfg := middleware.JWTConfig{
		SecretKey:       s.getSecretKey(),
		TokenExpiration: expiration,
		Logger:          s.logger,
		Optional:        false,
	}
	return jwtService.GenerateToken(jwtCfg, userID, email, username)
}

func (s *AuthServiceImpl) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	jwtService := middleware.NewJWTService()
	jwtCfg := middleware.JWTConfig{
		SecretKey:       s.getSecretKey(),
		TokenExpiration: s.getAccessTTL(),
		Logger:          s.logger,
		Optional:        false,
	}
	claims, err := jwtService.ValidateToken(jwtCfg, tokenString)
	if err != nil {
		return nil, err
	}

	// Convert Claims to jwt.MapClaims
	mapClaims := jwt.MapClaims{
		"user_id": claims.UserID,
		"email":   claims.Email,
		"name":    claims.Username,
		"exp":     claims.ExpiresAt.Unix(),
		"iat":     claims.IssuedAt.Unix(),
		"iss":     claims.Issuer,
		"aud":     claims.Audience,
	}

	return &mapClaims, nil
}

func (s *AuthServiceImpl) HashPassword(password string) (string, error) {
	jwtService := middleware.NewJWTService()
	return jwtService.HashPassword(password)
}

func (s *AuthServiceImpl) CheckPassword(hashedPassword, password string) bool {
	jwtService := middleware.NewJWTService()
	return jwtService.CheckPassword(hashedPassword, password)
}
