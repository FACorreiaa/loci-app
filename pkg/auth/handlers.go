package auth

import (
	"net/http"
	"time"

	"github.com/FACorreiaa/go-templui/pkg/logger"
	"go.uber.org/zap"
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
	authService *AuthService
}

func NewAuthHandlers() *AuthHandlers {
	return &AuthHandlers{
		authService: NewAuthService(),
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
		w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Email and password are required</div>`))
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

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
		return
	}

	logger.Log.Warn("Invalid login credentials",
		zap.String("email", email),
	)
	w.Header().Set("HX-Retarget", "#login-form")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Invalid email or password</div>`))
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

	firstName := r.FormValue("first-name")
	lastName := r.FormValue("last-name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm-password")

	if firstName == "" || lastName == "" || email == "" || password == "" {
		w.Header().Set("HX-Retarget", "#register-form")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="text-red-500 text-sm mb-4">All required fields must be filled</div>`))
		return
	}

	if password != confirmPassword {
		w.Header().Set("HX-Retarget", "#register-form")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="text-red-500 text-sm mb-4">Passwords do not match</div>`))
		return
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(password)
	if err != nil {
		logger.Log.Error("Failed to hash password", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate token for new user
	fullName := firstName + " " + lastName
	token, err := h.authService.GenerateToken("new-user-id", email, fullName)
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
		zap.String("hashed_password", hashedPassword), // Note: Don't log this in production
	)

	w.Header().Set("HX-Redirect", "/")
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
		w.Write([]byte(`<div class="text-red-500 text-sm mb-4">All fields are required</div>`))
		return
	}

	if newPassword != confirmNewPassword {
		w.Header().Set("HX-Retarget", "#change-password-form")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="text-red-500 text-sm mb-4">New passwords do not match</div>`))
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
	w.Write([]byte(`<div class="text-green-500 text-sm mb-4">Password changed successfully</div>`))
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

	http.Redirect(w, r, "/", http.StatusFound)
}