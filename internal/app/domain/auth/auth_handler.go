package auth

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/components/banner"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	FirstName       string `json:"first-name"`
	LastName        string `json:"last-name"`
	Email           string `json:"email"`
	Company         string `json:"company"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm-password"`
	Terms           bool   `json:"terms"`
}

type ChangePasswordRequest struct {
	CurrentPassword    string `json:"current-password"`
	NewPassword        string `json:"new-password"`
	ConfirmNewPassword string `json:"confirm-new-password"`
}

type AuthHandlers struct {
	authService AuthService
	logger      *zap.Logger
}

func NewAuthHandlers(authService AuthService, logger *zap.Logger) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
		logger:      logger,
	}
}

func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Login attempt",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	rememberMe := r.FormValue("remember_me") == "on" || r.FormValue("remember_me") == "true"

	if email == "" || password == "" {
		h.logger.Warn("Missing email or password")
		w.Header().Set("HX-Retarget", "#login-form")
		w.WriteHeader(http.StatusBadRequest)
		component := banner.Banner(banner.BannerProps{
			Type:        banner.BannerError,
			Message:     "Email and password are required",
			Dismissable: true,
			ID:          "login-error",
			AutoDismiss: 5,
		})
		if err := component.Render(r.Context(), w); err != nil {
			h.logger.Error("Failed to render banner", zap.Error(err))
		}
		return
	}

	// Validate credentials
	user, err := h.authService.GetUserByEmail(r.Context(), email)
	if err != nil || user == nil || !h.authService.CheckPassword(user.Password, password) {
		h.logger.Warn("Invalid login credentials",
			zap.String("email", email),
		)
		w.Header().Set("HX-Retarget", "#login-form")
		w.WriteHeader(http.StatusUnauthorized)
		component := banner.Banner(banner.BannerProps{
			Type:        banner.BannerError,
			Message:     "Invalid email or password",
			Description: "Please check your credentials and try again",
			Dismissable: true,
			ID:          "login-invalid",
			AutoDismiss: 5,
		})
		if err := component.Render(r.Context(), w); err != nil {
			h.logger.Error("Failed to render banner", zap.Error(err))
		}
		return
	}

	// Generate token with appropriate expiration based on remember me
	var tokenExpiration time.Duration
	var cookieMaxAge int

	if rememberMe {
		// Remember me: 30 days
		tokenExpiration = 30 * 24 * time.Hour
		cookieMaxAge = int(tokenExpiration.Seconds())
	} else {
		// Session only: 24 hours
		tokenExpiration = 24 * time.Hour
		cookieMaxAge = int(tokenExpiration.Seconds())
	}

	token, err := h.authService.GenerateTokenWithExpiration(user.ID, user.Email, user.Username, tokenExpiration)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set cookie with appropriate expiration
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}

	http.SetCookie(w, cookie)

	h.logger.Info("Successful login",
		zap.String("user_id", user.ID),
		zap.String("email", email),
		zap.Bool("remember_me", rememberMe),
		zap.Duration("token_expiration", tokenExpiration),
	)

	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Registration attempt",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	firstName := r.FormValue("firstname")
	lastName := r.FormValue("lastname")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if firstName == "" || lastName == "" || email == "" || password == "" || confirmPassword == "" {
		w.Header().Set("HX-Retarget", "#signup-response")
		w.WriteHeader(http.StatusBadRequest)
		component := banner.Banner(banner.BannerProps{
			Type:        banner.BannerError,
			Message:     "All required fields must be filled",
			Dismissable: true,
			ID:          "signup-required",
			AutoDismiss: 5,
		})
		if err := component.Render(r.Context(), w); err != nil {
			h.logger.Error("Failed to render banner", zap.Error(err))
		}
		return
	}

	if password != confirmPassword {
		w.Header().Set("HX-Retarget", "#signup-response")
		w.WriteHeader(http.StatusBadRequest)
		component := banner.Banner(banner.BannerProps{
			Type:        banner.BannerError,
			Message:     "Passwords do not match",
			Description: "Please ensure both password fields are identical",
			Dismissable: true,
			ID:          "signup-password-mismatch",
			AutoDismiss: 5,
		})
		if err := component.Render(r.Context(), w); err != nil {
			h.logger.Error("Failed to render banner", zap.Error(err))
		}
		return
	}

	// Register user in database and get user ID
	fullName := firstName + " " + lastName
	userID, err := h.authService.Register(r.Context(), fullName, email, password, "user")
	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err))
		w.Header().Set("HX-Retarget", "#signup-response")
		w.WriteHeader(http.StatusBadRequest)
		component := banner.Banner(banner.BannerProps{
			Type:        banner.BannerError,
			Message:     "Registration failed",
			Description: "Email may already be registered. Please try signing in or use a different email.",
			Dismissable: true,
			ID:          "signup-failed",
			AutoDismiss: 8,
		})
		if err := component.Render(r.Context(), w); err != nil {
			h.logger.Error("Failed to render banner", zap.Error(err))
		}
		return
	}

	// Generate token for new user using userID
	// Default to 24 hours for new registrations
	tokenExpiration := 24 * time.Hour
	token, err := h.authService.GenerateTokenWithExpiration(userID, email, fullName, tokenExpiration)
	if err != nil {
		h.logger.Error("Failed to generate token", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set cookie with appropriate expiration
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   int(tokenExpiration.Seconds()),
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	h.logger.Info("Successful registration",
		zap.String("user_id", userID),
		zap.String("email", email),
		zap.String("name", fullName),
	)

	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Change password attempt",
		zap.String("method", r.Method),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	currentPassword := r.FormValue("current-password")
	newPassword := r.FormValue("new-password")
	confirmNewPassword := r.FormValue("confirm-new-password")

	if currentPassword == "" || newPassword == "" || confirmNewPassword == "" {
		w.Header().Set("HX-Retarget", "#change-password-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">All fields are required</div>`)); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	if newPassword != confirmNewPassword {
		w.Header().Set("HX-Retarget", "#change-password-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">New passwords do not match</div>`)); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Hash new password
	_, err = h.authService.HashPassword(newPassword)
	if err != nil {
		h.logger.Error("Failed to hash new password", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Password changed successfully")

	w.Header().Set("HX-Retarget", "#change-password-form")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<div class="text-green-500 text-sm mb-4">Password changed successfully</div>`)); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
	}
}

func (h *AuthHandlers) LogoutHandler(w http.ResponseWriter, _ *http.Request) {
	h.logger.Info("User logout")

	// Clear auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// For HTMX requests, redirect to home page
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandlers) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Password reset request",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		w.Header().Set("HX-Retarget", "#forgot-password-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Email is required</div>`)); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Always show success message for security (don't reveal if email exists)
	h.logger.Info("Password reset requested", zap.String("email", email))
	w.Header().Set("HX-Retarget", "#forgot-password-form")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<div class="text-green-500 text-sm mb-4">If this email is registered, you will receive password reset instructions</div>`)); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
	}
}

func (h *AuthHandlers) CheckUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		h.logger.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<span class="text-red-500 text-sm">Username is required</span>`)); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Simple validation for demo - in production, check against database
	if len(username) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<span class="text-red-500 text-sm">Username must be at least 3 characters</span>`)); err != nil {
			h.logger.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// For demo, assume username is available
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<span class="text-green-500 text-sm">Username is available</span>`)); err != nil {
		h.logger.Error("Failed to write response", zap.Error(err))
	}
}
