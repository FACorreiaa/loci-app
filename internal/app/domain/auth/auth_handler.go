package auth

import (
	"errors"
	"net/http"
	"net/mail"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/FACorreiaa/go-templui/internal/app/components/banner"
	"github.com/FACorreiaa/go-templui/internal/app/domain"
	"github.com/FACorreiaa/go-templui/internal/app/domain/pages"
	"github.com/FACorreiaa/go-templui/internal/app/models"
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
	*domain.BaseHandler
	authService AuthService
	logger      *zap.Logger
}

func NewAuthHandlers(base *domain.BaseHandler, authService AuthService, logger *zap.Logger) *AuthHandlers {
	return &AuthHandlers{
		BaseHandler: base,
		authService: authService,
		logger:      logger,
	}
}

// #Login
// LoginHandler
func (h *AuthHandlers) LoginHandler(c *gin.Context) {
	h.logger.Info("Login attempt",
		zap.String("method", c.Request.Method),
		zap.String("remote_addr", c.Request.RemoteAddr),
	)

	email := c.PostForm("email")
	password := c.PostForm("password")
	rememberMe := c.PostForm("remember_me") == "on"

	formErrors := SignInFormErrors{}
	if email == "" {
		formErrors.Email = "Email address is required."
	} else {
		_, err := mail.ParseAddress(email)
		if err != nil {
			formErrors.Email = "Please enter a valid email address."
		}
	}
	if password == "" {
		formErrors.Password = "Password is required."
	}

	if formErrors.Email != "" || formErrors.Password != "" {
		h.logger.Warn("Validation failed on login", zap.Any("errors", formErrors))
		formValues := SignInFormValues{Email: email}
		component := SignInForm(formValues, formErrors)
		c.Status(http.StatusOK)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	accessToken, refreshToken, err := h.authService.Login(c.Request.Context(), email, password)
	if err != nil {
		h.logger.Warn("Invalid login credentials", zap.String("email", email), zap.Error(err))
		formValues := SignInFormValues{Email: email}
		formErrors.General = "Invalid email or password. Please try again."

		switch {
		case errors.Is(err, models.ErrUserNotFound), errors.Is(err, models.ErrInvalidPassword):
			formErrors.General = "Invalid credentials. Please check your email and password."
		default:
			formErrors.General = "An unexpected error occurred. Please try again."
		}

		component := SignInForm(formValues, formErrors)
		c.Status(http.StatusOK)
		component.Render(c.Request.Context(), c.Writer)
		return
	}

	var tokenExpiration time.Duration
	if rememberMe {
		tokenExpiration = 30 * 24 * time.Hour // 30 days
	} else {
		tokenExpiration = 24 * time.Hour // 24 hours (session)
	}
	cookieMaxAge := int(tokenExpiration.Seconds())

	c.SetCookie(
		"auth_token", // name
		accessToken,  // value
		cookieMaxAge, // maxAge
		"/",          // path
		"",           // domain (blank for current)
		false,        // secure (set true in production)
		true,         // httpOnly
	)

	c.SetCookie(
		"refresh_token",
		refreshToken,
		cookieMaxAge*30, // Example: much longer life
		"/",
		"",
		false,
		true,
	)

	h.logger.Info("Successful login",
		zap.String("email", email),
		zap.Bool("remember_me", rememberMe),
	)

	c.Header("HX-Redirect", "/dashboard")
	c.Status(http.StatusOK)
}

func (h *AuthHandlers) ShowSignInPage(c *gin.Context) {
	h.Logger.Info("Sign in page accessed")

	initialValues := SignInFormValues{}
	initialErrors := SignInFormErrors{}
	//var err error
	//switch {
	//case errors.Is(err, models.ErrInvalidEmailFormat):
	//	initialErrors.Email = "Please enter a valid email address."
	//case errors.Is(err, models.ErrUserNotFound):
	//	initialErrors.General = "Invalid credentials. Please check your email and password."
	//case errors.Is(err, models.ErrInvalidPassword):
	//	initialErrors.General = "Invalid credentials. Please check your email and password."
	//default:
	//	initialErrors.General = ""
	//}

	content := SignIn(initialValues, initialErrors)
	layoutData := h.NewLayoutData(c, "SignIn - Loci", "SignIn", content)
	layoutData.Nav = models.OfflineNav
	layoutData.User = nil

	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
}

func (h *AuthHandlers) ShowSignUpPage(c *gin.Context) {
	h.Logger.Info("Sign up page accessed")

	content := SignUp()
	layoutData := h.NewLayoutData(c, "Signup - Loci", "Signup", content)
	layoutData.Nav = models.OfflineNav
	layoutData.User = nil

	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
}

func (h *AuthHandlers) ShowForgotPasswordPage(c *gin.Context) {
	h.Logger.Info("Forgot password page accessed")

	content := ForgotPassword()
	layoutData := h.NewLayoutData(c, "Forgot password - Loci", "Forgot password", content)
	layoutData.Nav = models.OfflineNav
	layoutData.User = nil

	h.Render(c, http.StatusOK, pages.LayoutPage(layoutData))
}

// #Register
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
