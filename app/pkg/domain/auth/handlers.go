package auth

import (
	"log/slog"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/app/pkg/config"
	"github.com/FACorreiaa/go-templui/app/pkg/logger"
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
}

func NewAuthHandlers(repo AuthRepo, cfg *config.Config, logger *slog.Logger) *AuthHandlers {
	return &AuthHandlers{
		authService: NewAuthService(repo, cfg, logger),
	}
}

func (h *AuthHandlers) LoginHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Login attempt",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "" || password == "" {
		logger.Log.Warn("Missing email or password")
		w.Header().Set("HX-Retarget", "#login-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Email and password are required</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Demo user for testing
	if email == "demo@crunchbase.com" && password == "password123" {
		token, err := h.authService.GenerateToken("demo-user-id", email, "Demo User")
		if err != nil {
			logger.Log.Error("Failed to generate token", zap.Error(err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		})

		logger.Log.Info("Successful login",
			zap.String("email", email),
		)

		w.Header().Set("HX-Redirect", "/dashboard")
		w.WriteHeader(http.StatusOK)
		return
	}

	logger.Log.Warn("Invalid login credentials",
		zap.String("email", email),
	)
	w.Header().Set("HX-Retarget", "#login-form")
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Invalid email or password</div>`)); err != nil {
		logger.Log.Error("Failed to write response", zap.Error(err))
	}
}

func (h *AuthHandlers) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Registration attempt",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	firstName := r.FormValue("firstname")
	lastName := r.FormValue("lastname")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if firstName == "" || lastName == "" || email == "" || password == "" {
		w.Header().Set("HX-Retarget", "#register-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">All required fields must be filled</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	if password != confirmPassword {
		w.Header().Set("HX-Retarget", "#register-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Passwords do not match</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Register user in database
	fullName := firstName + " " + lastName
	err = h.authService.Register(r.Context(), fullName, email, password, "user")
	if err != nil {
		logger.Log.Error("Failed to register user", zap.Error(err))
		w.Header().Set("HX-Retarget", "#register-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Registration failed. Email may already be registered.</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Generate token for new user
	userID := email // Use email as unique identifier
	token, err := h.authService.GenerateToken(userID, email, fullName)
	if err != nil {
		logger.Log.Error("Failed to generate token", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	logger.Log.Info("Successful registration",
		zap.String("email", email),
		zap.String("name", fullName),
	)

	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("Change password attempt",
		zap.String("method", r.Method),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Failed to parse form", zap.Error(err))
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
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	if newPassword != confirmNewPassword {
		w.Header().Set("HX-Retarget", "#change-password-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">New passwords do not match</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Hash new password
	_, err = h.authService.HashPassword(newPassword)
	if err != nil {
		logger.Log.Error("Failed to hash new password", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.Log.Info("Password changed successfully")

	w.Header().Set("HX-Retarget", "#change-password-form")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<div class="text-green-500 text-sm mb-4">Password changed successfully</div>`)); err != nil {
		logger.Log.Error("Failed to write response", zap.Error(err))
	}
}

func (h *AuthHandlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	logger.Log.Info("User logout")

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
	logger.Log.Info("Password reset request",
		zap.String("method", r.Method),
		zap.String("remote_addr", r.RemoteAddr),
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		w.Header().Set("HX-Retarget", "#forgot-password-form")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Email is required</div>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Always show success message for security (don't reveal if email exists)
	logger.Log.Info("Password reset requested", zap.String("email", email))
	w.Header().Set("HX-Retarget", "#forgot-password-form")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<div class="text-green-500 text-sm mb-4">If this email is registered, you will receive password reset instructions</div>`)); err != nil {
		logger.Log.Error("Failed to write response", zap.Error(err))
	}
}

func (h *AuthHandlers) CheckUsernameHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		logger.Log.Error("Failed to parse form", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<span class="text-red-500 text-sm">Username is required</span>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// Simple validation for demo - in production, check against database
	if len(username) < 3 {
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`<span class="text-red-500 text-sm">Username must be at least 3 characters</span>`)); err != nil {
			logger.Log.Error("Failed to write response", zap.Error(err))
		}
		return
	}

	// For demo, assume username is available
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<span class="text-green-500 text-sm">Username is available</span>`)); err != nil {
		logger.Log.Error("Failed to write response", zap.Error(err))
	}
}
