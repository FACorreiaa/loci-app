package auth

import "errors"

var (
	ErrWrongCredentials = errors.New("invalid login credentials")
)

// Error messages for validation and authentication
const (
	// Login errors
	MsgEmailRequired          = "Email address is required."
	MsgEmailInvalid           = "Please enter a valid email address."
	MsgPasswordRequired       = "Password is required."
	MsgPasswordInvalid        = "Please enter a valid password."
	MsgInvalidCredentials     = "Invalid credentials. Please check your email and password."
	MsgLoginFailed            = "Invalid email or password. Please try again."
	MsgUnexpectedError        = "An unexpected error occurred. Please try again."

	// Registration errors
	MsgAllFieldsRequired      = "All required fields must be filled"
	MsgPasswordMismatch       = "Passwords do not match"
	MsgPasswordMismatchDesc   = "Please ensure both password fields are identical"
	MsgRegistrationFailed     = "Registration failed"
	MsgEmailAlreadyRegistered = "Email may already be registered. Please try signing in or use a different email."

	// Change password errors
	MsgNewPasswordMismatch    = "New passwords do not match"
	MsgPasswordChangeSuccess  = "Password changed successfully"

	// Forgot password messages
	MsgEmailRequiredForgot    = "Email is required"
	MsgPasswordResetSent      = "If this email is registered, you will receive password reset instructions"

	// Username validation
	MsgUsernameRequired       = "Username is required"
	MsgUsernameTooShort       = "Username must be at least 3 characters"
	MsgUsernameAvailable      = "Username is available"
)
