# Templ Authentication System Documentation

## Overview

This document describes the comprehensive authentication system implemented for the Loci TemplUI application using Templ templates, HTMX for dynamic interactions, and proper JWT security practices.

## Architecture

### Components

1. **Templates** (`app/lib/features/auth/`)
   - `signin.templ` - Sign-in form with HTMX integration
   - `signup.templ` - User registration form
   - `forgot-password.templ` - Password reset request form

2. **Handlers** (`app/pkg/domain/auth/`)
   - `enhanced_handlers.go` - Enhanced auth handlers with proper validation
   - `handlers.go` - Original handlers (legacy)
   - `auth_service.go` - Business logic layer
   - `auth_repository.go` - Database access layer

3. **Models** (`app/lib/models/`)
   - `auth.go` - Authentication-related data structures
   - `user.go` - User model definitions

4. **Configuration** (`app/pkg/config/`)
   - `config.go` - JWT configuration and security settings

## Features

### 1. Sign In

**Template**: `signin.templ`
**Handler**: `LoginHandler`
**Endpoint**: `POST /auth/signin`

#### Features:
- Email and password validation
- HTMX-powered form submission
- Password visibility toggle
- "Remember me" functionality
- Rate limiting protection
- Secure cookie management
- Social login integration (Google, GitHub)

#### HTMX Integration:
```html
<form hx-post="/auth/signin" hx-trigger="submit" hx-target="#signin-response" hx-indicator="#signin-loading">
```

#### Security Features:
- Input sanitization
- CSRF protection via HTMX headers
- Secure HTTP-only cookies
- Rate limiting (client-side and server-side)
- Generic error messages to prevent user enumeration

### 2. Sign Up

**Template**: `signup.templ`
**Handler**: `RegisterHandler`
**Endpoint**: `POST /auth/signup`

#### Features:
- Comprehensive form validation
- Real-time username availability checking
- Password strength indicator
- Terms of service agreement
- Marketing consent opt-in
- HTMX form submission

#### Validation:
- **Email**: Format validation using regex
- **Password**: Minimum 8 characters, uppercase, lowercase, digit
- **Names**: 2-50 characters, letters/spaces/hyphens/apostrophes only
- **Username**: 3-20 characters, alphanumeric/underscore/hyphen only

#### HTMX Integration:
```html
<form hx-post="/auth/signup" hx-trigger="submit" hx-target="#signup-response">
```

### 3. Password Reset

**Template**: `forgot-password.templ`
**Handler**: `ForgotPasswordHandler`
**Endpoint**: `POST /auth/forgot-password`

#### Features:
- Email-based password reset
- Security-focused UX (always shows success)
- Rate limiting
- CAPTCHA integration
- Resend functionality with cooldown

#### Security Considerations:
- Always returns success to prevent email enumeration
- Rate limiting per IP address
- Secure token generation (to be implemented)
- Email validation

## JWT Implementation

### Configuration

JWT settings are configured in `config.go`:

```go
type JWTConfig struct {
    SecretKey       string
    Issuer          string
    Audience        string
    AccessTokenTTL  time.Duration
    RefreshTokenTTL time.Duration
}
```

### Best Practices Implemented

1. **Token Types**:
   - **Access Token**: Short-lived (15 minutes), used for API requests
   - **Refresh Token**: Longer-lived (7 days with "remember me"), used to obtain new access tokens

2. **Secure Storage**:
   - HTTP-only cookies prevent XSS attacks
   - Secure flag ensures HTTPS-only transmission
   - SameSite=Strict prevents CSRF attacks

3. **Token Rotation**:
   - Refresh tokens are invalidated after use
   - All user tokens can be invalidated for security

4. **Cookie Configuration**:
   ```go
   accessCookie := &http.Cookie{
       Name:     "access_token",
       Value:    accessToken,
       Path:     "/",
       HttpOnly: true,
       Secure:   true,
       SameSite: http.SameSiteStrictMode,
       MaxAge:   15 * 60, // 15 minutes
   }
   ```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### User Providers Table (OAuth)
```sql
CREATE TABLE user_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
```

## HTMX Integration

### Form Submission
All auth forms use HTMX for seamless submission without page reloads:

```html
<form hx-post="/auth/signin" 
      hx-trigger="submit" 
      hx-target="#signin-response" 
      hx-indicator="#signin-loading"
      hx-headers='{"X-Requested-With": "XMLHttpRequest"}'>
```

### Response Handling
Handlers detect HTMX requests and respond appropriately:

```go
func (h *EnhancedAuthHandlers) sendErrorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
    if r.Header.Get("HX-Request") == "true" {
        // Send HTMX-compatible error response
        errorHTML := fmt.Sprintf(`<div class="text-red-500 text-sm bg-red-50 p-3 rounded-lg border border-red-200">%s</div>`, message)
        _, _ = w.Write([]byte(errorHTML))
    } else {
        // Send JSON for non-HTMX requests
        json.NewEncoder(w).Encode(map[string]interface{}{
            "error":   true,
            "message": message,
            "status":  status,
        })
    }
}
```

### Redirects
HTMX redirects are handled using the `HX-Redirect` header:

```go
if r.Header.Get("HX-Request") == "true" {
    w.Header().Set("HX-Redirect", "/dashboard")
    h.sendSuccessResponse(w, r, "Login successful! Redirecting...")
}
```

## Security Features

### Input Validation
All inputs are validated both client-side and server-side:

```go
func (h *EnhancedAuthHandlers) validateEmail(email string) error {
    if email == "" {
        return fmt.Errorf("email is required")
    }
    
    emailRegex := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
    if !emailRegex.MatchString(email) {
        return fmt.Errorf("invalid email format")
    }
    
    return nil
}
```

### Password Security
- Minimum requirements: 8+ characters, uppercase, lowercase, digit
- Passwords are hashed using bcrypt with appropriate cost
- Password visibility toggle for UX

### Rate Limiting
- Client-side rate limiting with localStorage tracking
- Server-side rate limiting (to be implemented with middleware)
- Exponential backoff for failed attempts

### CSRF Protection
- HTMX requests include `X-Requested-With` header
- SameSite cookie protection
- Double-submit cookie pattern (can be added)

### Session Management
- Secure cookie configuration
- Token expiration and rotation
- Graceful logout with token invalidation

## Testing

### Unit Tests
- **Handler Tests** (`enhanced_handlers_test.go`): Mock-based testing of all handlers
- **Template Tests** (`auth_integration_test.go`): Template rendering and structure validation
- **Validation Tests**: Input validation function testing

### Test Coverage
- Login flow (success, failure, validation errors)
- Registration flow (validation, conflicts, success)
- Logout flow (with/without tokens)
- Username availability checking
- Form validation (email, password, names, username)

### Example Test
```go
func TestEnhancedAuthHandlers_LoginHandler(t *testing.T) {
    handler, mockService := createTestHandler()
    
    tests := []struct {
        name           string
        method         string
        formData       url.Values
        setupMock      func(*MockAuthService)
        expectedStatus int
        expectedBody   string
        isHTMX         bool
    }{
        {
            name:   "successful login",
            method: http.MethodPost,
            formData: url.Values{
                "email":    {"test@example.com"},
                "password": {"password123"},
            },
            setupMock: func(m *MockAuthService) {
                m.On("Login", mock.Anything, "test@example.com", "password123").
                    Return("access-token", "refresh-token", nil)
            },
            expectedStatus: http.StatusOK,
            expectedBody:   "Login successful",
            isHTMX:         true,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## API Endpoints

| Method | Endpoint | Description | HTMX |
|--------|----------|-------------|------|
| GET | `/auth/signin` | Sign in page | ✓ |
| POST | `/auth/signin` | Process login | ✓ |
| GET | `/auth/signup` | Sign up page | ✓ |
| POST | `/auth/signup` | Process registration | ✓ |
| GET | `/auth/forgot-password` | Password reset page | ✓ |
| POST | `/auth/forgot-password` | Process reset request | ✓ |
| POST | `/auth/logout` | User logout | ✓ |
| POST | `/auth/change-password` | Change password | ✓ |
| POST | `/auth/check-username` | Check availability | ✓ |

## Error Handling

### Client-Side Errors
- Real-time validation feedback
- User-friendly error messages
- Visual indicators (red borders, error text)

### Server-Side Errors
- Structured error responses
- Logging with context
- Generic messages for security
- Proper HTTP status codes

### Example Error Response
```html
<div class="text-red-500 text-sm bg-red-50 p-3 rounded-lg border border-red-200">
    Invalid email or password
</div>
```

## Styling and UX

### CSS Framework
- Tailwind CSS for styling
- Responsive design (mobile-first)
- Dark mode support
- Consistent component styling

### User Experience
- Loading indicators during requests
- Smooth transitions and animations
- Clear visual feedback
- Accessibility compliant (WCAG)

### Design Patterns
- Card-based layout for forms
- Consistent spacing and typography
- Color-coded feedback (red=error, green=success)
- Progressive enhancement

## Future Enhancements

### Planned Features
1. **Email Verification**: Account activation via email
2. **Two-Factor Authentication**: TOTP/SMS support
3. **Social OAuth**: Complete Google/GitHub integration
4. **Password Reset**: Complete email-based reset flow
5. **Account Recovery**: Multiple recovery options
6. **Session Management**: Advanced session controls

### Security Improvements
1. **CAPTCHA Integration**: reCAPTCHA for bot prevention
2. **Device Fingerprinting**: Enhanced security monitoring
3. **Audit Logging**: Comprehensive auth event logging
4. **IP Whitelisting**: Location-based security
5. **Breach Monitoring**: Password breach detection

### Performance Optimizations
1. **Caching**: Redis for session storage
2. **CDN**: Static asset delivery
3. **Minification**: CSS/JS optimization
4. **Lazy Loading**: Component optimization

## Troubleshooting

### Common Issues

1. **HTMX Not Working**
   - Check HTMX script inclusion
   - Verify content-type headers
   - Check browser console for errors

2. **JWT Token Issues**
   - Verify secret key configuration
   - Check token expiration times
   - Validate cookie settings

3. **Database Connection**
   - Verify PostgreSQL connection
   - Check migration status
   - Validate table schemas

4. **Validation Errors**
   - Check regex patterns
   - Verify input sanitization
   - Test with various inputs

### Debugging

1. **Enable Debug Logging**:
   ```go
   logger.Log.Debug("Authentication attempt", 
       zap.String("email", email),
       zap.String("ip", r.RemoteAddr))
   ```

2. **HTMX Debug Mode**:
   ```html
   <script>
   htmx.config.historyCacheSize = 0; // Disable cache
   htmx.logAll(); // Enable all logging
   </script>
   ```

3. **Database Query Logging**:
   Enable PostgreSQL query logging for debugging.

## Conclusion

This authentication system provides a secure, user-friendly, and modern auth experience using Templ templates with HTMX integration. The implementation follows security best practices while maintaining excellent user experience through progressive enhancement and responsive design.

The system is designed to be extensible and maintainable, with comprehensive testing coverage and clear separation of concerns between presentation, business logic, and data access layers.