# HTMX Authentication Redirect Fix

## Problem
When a user logged out or the session was lost on the discover page, the login form was being displayed inline within the discover page instead of redirecting the user to the login page. This was an HTMX-specific issue.

## Root Cause
The `AuthMiddleware` in `/internal/pkg/middleware/middleware.go` was using standard HTTP redirects (`c.Redirect(http.StatusFound, "/auth/signin")`) for all requests, including HTMX requests.

When HTMX makes a request, it expects special headers to trigger client-side redirects. If you return a standard HTTP redirect, HTMX will follow it and swap the response HTML into the target element, resulting in the login form appearing inside the page instead of a full page navigation.

## Solution
Modified the `AuthMiddleware` to detect HTMX requests and handle them appropriately:

1. **Created a new helper function** `handleAuthRedirect` that:
   - Checks if the request is an HTMX request by looking for the `HX-Request` header
   - For HTMX requests: Sets the `HX-Redirect` header and returns `401 Unauthorized`
   - For regular requests: Uses standard HTTP redirect with `302 Found`

2. **Updated AuthMiddleware** to use the new helper function instead of direct redirects

## Code Changes

### File: `/internal/pkg/middleware/middleware.go`

```go
// AuthMiddleware validates authentication tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("auth_token")
		if err != nil {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// Validate JWT token
		if token == "" {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// ... validation logic ...

		claims, err := jwtService.ValidateToken(config, token)
		if err != nil {
			handleAuthRedirect(c, "/auth/signin")
			return
		}

		// Set user context from JWT claims
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Username)
		c.Next()
	}
}

// handleAuthRedirect handles redirects for both regular and HTMX requests
func handleAuthRedirect(c *gin.Context, redirectURL string) {
	// Check if this is an HTMX request
	if c.GetHeader("HX-Request") == "true" {
		// For HTMX requests, use HX-Redirect header to trigger client-side redirect
		c.Header("HX-Redirect", redirectURL)
		c.AbortWithStatus(http.StatusUnauthorized)
	} else {
		// For regular requests, use standard HTTP redirect
		c.Redirect(http.StatusFound, redirectURL)
		c.Abort()
	}
}
```

## How It Works

### HTMX Request Flow:
1. User makes an HTMX request (e.g., search on discover page)
2. Middleware detects missing/invalid auth token
3. Middleware detects `HX-Request: true` header
4. Middleware sets `HX-Redirect: /auth/signin` header
5. Returns `401 Unauthorized` status
6. HTMX client-side JavaScript sees the `HX-Redirect` header
7. HTMX triggers a full page redirect to `/auth/signin`

### Regular Request Flow:
1. User makes a regular HTTP request
2. Middleware detects missing/invalid auth token
3. Middleware does NOT see `HX-Request` header
4. Middleware returns standard `302 Found` redirect
5. Browser follows redirect to `/auth/signin`

## Testing
Created unit test in `middleware_test.go` that verifies:
- HTMX requests get `HX-Redirect` header and `401` status
- Regular requests get standard `Location` header and `302` status

Test output:
```
=== RUN   TestAuthMiddlewareHTMXRedirect
=== RUN   TestAuthMiddlewareHTMXRedirect/HTMX_request_without_auth_should_get_HX-Redirect_header
=== RUN   TestAuthMiddlewareHTMXRedirect/Regular_request_without_auth_should_get_HTTP_redirect
--- PASS: TestAuthMiddlewareHTMXRedirect (0.00s)
    --- PASS: TestAuthMiddlewareHTMXRedirect/HTMX_request_without_auth_should_get_HX-Redirect_header (0.00s)
    --- PASS: TestAuthMiddlewareHTMXRedirect/Regular_request_without_auth_should_get_HTTP_redirect (0.00s)
PASS
```

## Benefits
- Users are properly redirected to the login page when their session expires
- No more login form appearing inline within other pages
- Works correctly for both HTMX and non-HTMX requests
- Follows HTMX best practices for handling redirects
- Maintains security by requiring authentication for protected routes

## Related Files
- `/internal/pkg/middleware/middleware.go` - Main fix
- `/middleware_test.go` - Unit tests for the fix
- `/internal/app/domain/auth/auth_handler.go` - Logout handler (already uses `HX-Redirect`)

## Additional Notes
This same pattern should be applied to any other middleware that needs to redirect users, such as:
- Role-based access control middleware
- Subscription/plan verification middleware
- Any custom authorization middleware
