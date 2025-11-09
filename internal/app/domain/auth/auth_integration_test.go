package auth_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/a-h/templ"

	"github.com/FACorreiaa/go-templui/internal/app/domain/auth"
)

func TestSignInTemplate(t *testing.T) {
	tests := []struct {
		name   string
		assert func(*testing.T, *goquery.Document)
	}{
		{
			name: "renders signin form with HTMX attributes",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check form exists with HTMX attributes
				form := doc.Find("form[hx-post='/auth/signin']")
				if form.Length() == 0 {
					t.Error("Expected signin form with hx-post='/auth/signin' to be rendered")
				}

				// Check HTMX attributes
				htmxTrigger, _ := form.Attr("hx-trigger")
				if htmxTrigger != "submit" {
					t.Errorf("Expected hx-trigger='submit', got '%s'", htmxTrigger)
				}

				htmxTarget, _ := form.Attr("hx-target")
				if htmxTarget != "#signin-response" {
					t.Errorf("Expected hx-target='#signin-response', got '%s'", htmxTarget)
				}

				htmxIndicator, _ := form.Attr("hx-indicator")
				if htmxIndicator != "#signin-loading" {
					t.Errorf("Expected hx-indicator='#signin-loading', got '%s'", htmxIndicator)
				}
			},
		},
		{
			name: "includes required form fields",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check email field
				emailInput := doc.Find("input[name='email'][type='email']")
				if emailInput.Length() == 0 {
					t.Error("Expected email input field to be rendered")
				}

				// Check email field has required attribute
				if _, exists := emailInput.Attr("required"); !exists {
					t.Error("Email input should have required attribute")
				}

				// Check password field
				passwordInput := doc.Find("input[name='password'][type='password']")
				if passwordInput.Length() == 0 {
					t.Error("Expected password input field to be rendered")
				}

				// Check password field has required attribute
				if _, exists := passwordInput.Attr("required"); !exists {
					t.Error("Password input should have required attribute")
				}

				// Check remember me checkbox
				rememberCheckbox := doc.Find("input[name='remember_me'][type='checkbox']")
				if rememberCheckbox.Length() == 0 {
					t.Error("Expected remember me checkbox to be rendered")
				}
			},
		},
		{
			name: "includes response container for HTMX",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check for response container
				responseContainer := doc.Find("#signin-response")
				if responseContainer.Length() == 0 {
					t.Error("Expected #signin-response container to be rendered")
				}

				// Check for loading indicator
				loadingIndicator := doc.Find("#signin-loading")
				if loadingIndicator.Length() == 0 {
					t.Error("Expected #signin-loading indicator to be rendered")
				}
			},
		},
		{
			name: "includes navigation links",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check forgot password link
				forgotLink := doc.Find("a[href='/auth/forgot-password']")
				if forgotLink.Length() == 0 {
					t.Error("Expected forgot password link to be rendered")
				}

				// Check signup link
				signupLink := doc.Find("a[href='/auth/signup']")
				if signupLink.Length() == 0 {
					t.Error("Expected signup link to be rendered")
				}
			},
		},
		{
			name: "includes password visibility toggle",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check for password toggle function call
				script := doc.Find("script").Text()
				if !strings.Contains(script, "togglePasswordVisibility") {
					t.Error("Expected togglePasswordVisibility function to be defined")
				}

				// Check for toggle button
				toggleButton := doc.Find("button[onclick*='togglePasswordVisibility']")
				if toggleButton.Length() == 0 {
					t.Error("Expected password toggle button to be rendered")
				}
			},
		},
		{
			name: "includes submit button with proper text",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check submit button
				submitButton := doc.Find("button[type='submit']")
				if submitButton.Length() == 0 {
					t.Error("Expected submit button to be rendered")
				}

				// Check button text
				buttonText := strings.TrimSpace(submitButton.Text())
				if !strings.Contains(buttonText, "Sign in") {
					t.Errorf("Expected button to contain 'Sign in', got '%s'", buttonText)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Render the component
			r, w := io.Pipe()
			go func() {
				if err := auth.SignIn().Render(context.Background(), w); err != nil {
					t.Errorf("Failed to render SignIn: %v", err)
					return
				}
				_ = w.Close()
			}()

			// Parse with goquery
			doc, err := goquery.NewDocumentFromReader(r)
			if err != nil {
				t.Fatalf("failed to read template: %v", err)
			}

			// Run the test assertion
			test.assert(t, doc)
		})
	}
}

func TestSignUpTemplate(t *testing.T) {
	tests := []struct {
		name   string
		assert func(*testing.T, *goquery.Document)
	}{
		{
			name: "renders signup form with HTMX attributes",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check form exists with HTMX attributes
				form := doc.Find("form[hx-post='/auth/signup']")
				if form.Length() == 0 {
					t.Error("Expected signup form with hx-post='/auth/signup' to be rendered")
				}

				// Check HTMX attributes
				htmxTrigger, _ := form.Attr("hx-trigger")
				if htmxTrigger != "submit" {
					t.Errorf("Expected hx-trigger='submit', got '%s'", htmxTrigger)
				}

				htmxTarget, _ := form.Attr("hx-target")
				if htmxTarget != "#signup-response" {
					t.Errorf("Expected hx-target='#signup-response', got '%s'", htmxTarget)
				}
			},
		},
		{
			name: "includes all required registration fields",
			assert: func(t *testing.T, doc *goquery.Document) {
				requiredFields := map[string]string{
					"firstname": "text",
					"lastname":  "text",
					"email":     "email",
					"password":  "password",
				}

				for name, inputType := range requiredFields {
					input := doc.Find(fmt.Sprintf("input[name='%s'][type='%s']", name, inputType))
					if input.Length() == 0 {
						t.Errorf("Expected %s input field to be rendered", name)
					}

					// Check required attribute
					if _, exists := input.Attr("required"); !exists {
						t.Errorf("%s input should have required attribute", name)
					}
				}
			},
		},
		{
			name: "includes response container for HTMX",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check for response container
				responseContainer := doc.Find("#signup-response")
				if responseContainer.Length() == 0 {
					t.Error("Expected #signup-response container to be rendered")
				}

				// Check for loading indicator
				loadingIndicator := doc.Find("#signup-loading")
				if loadingIndicator.Length() == 0 {
					t.Error("Expected #signup-loading indicator to be rendered")
				}
			},
		},
		{
			name: "includes signin link",
			assert: func(t *testing.T, doc *goquery.Document) {
				signinLink := doc.Find("a[href='/auth/signin']")
				if signinLink.Length() == 0 {
					t.Error("Expected signin link to be rendered")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Render the component
			r, w := io.Pipe()
			go func() {
				if err := auth.SignUp().Render(context.Background(), w); err != nil {
					t.Errorf("Failed to render SignUp: %v", err)
					return
				}
				_ = w.Close()
			}()

			// Parse with goquery
			doc, err := goquery.NewDocumentFromReader(r)
			if err != nil {
				t.Fatalf("failed to read template: %v", err)
			}

			// Run the test assertion
			test.assert(t, doc)
		})
	}
}

func TestForgotPasswordTemplate(t *testing.T) {
	tests := []struct {
		name   string
		assert func(*testing.T, *goquery.Document)
	}{
		{
			name: "renders forgot password form with HTMX attributes",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check form exists with HTMX attributes
				form := doc.Find("form[hx-post='/auth/forgot-password']")
				if form.Length() == 0 {
					t.Error("Expected forgot password form with hx-post='/auth/forgot-password' to be rendered")
				}

				// Check HTMX target
				htmxTarget, _ := form.Attr("hx-target")
				if htmxTarget != "#forgot-response" {
					t.Errorf("Expected hx-target='#forgot-response', got '%s'", htmxTarget)
				}
			},
		},
		{
			name: "includes email field",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check email field
				emailInput := doc.Find("input[name='email'][type='email']")
				if emailInput.Length() == 0 {
					t.Error("Expected email input field to be rendered")
				}

				// Check required attribute
				if _, exists := emailInput.Attr("required"); !exists {
					t.Error("Email input should have required attribute")
				}
			},
		},
		{
			name: "includes response container",
			assert: func(t *testing.T, doc *goquery.Document) {
				// Check for response container
				responseContainer := doc.Find("#forgot-response")
				if responseContainer.Length() == 0 {
					t.Error("Expected #forgot-response container to be rendered")
				}
			},
		},
		{
			name: "includes back to signin link",
			assert: func(t *testing.T, doc *goquery.Document) {
				backLink := doc.Find("a[href='/auth/signin']")
				if backLink.Length() == 0 {
					t.Error("Expected back to signin link to be rendered")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Render the component
			r, w := io.Pipe()
			go func() {
				if err := auth.ForgotPassword().Render(context.Background(), w); err != nil {
					t.Errorf("Failed to render ForgotPassword: %v", err)
					return
				}
				_ = w.Close()
			}()

			// Parse with goquery
			doc, err := goquery.NewDocumentFromReader(r)
			if err != nil {
				t.Fatalf("failed to read template: %v", err)
			}

			// Run the test assertion
			test.assert(t, doc)
		})
	}
}

func TestAuthTemplatesAccessibility(t *testing.T) {
	templates := map[string]func() templ.Component{
		"signin": auth.SignIn,
		"signup": auth.SignUp,
		"forgot": auth.ForgotPassword,
	}

	for name, template := range templates {
		t.Run(name+"_accessibility", func(t *testing.T) {
			// Render the component
			r, w := io.Pipe()
			go func() {
				defer w.Close()
				if err := template().Render(context.Background(), w); err != nil {
					t.Errorf("Failed to render template: %v", err)
				}
			}()

			// Parse with goquery
			doc, err := goquery.NewDocumentFromReader(r)
			if err != nil {
				t.Fatalf("failed to read template: %v", err)
			}

			// Check for proper labels
			inputs := doc.Find("input[type='email'], input[type='password'], input[type='text']")
			inputs.Each(func(_ int, input *goquery.Selection) {
				inputID, exists := input.Attr("id")
				if exists {
					label := doc.Find(fmt.Sprintf("label[for='%s']", inputID))
					if label.Length() == 0 {
						t.Errorf("Input with ID '%s' should have a corresponding label", inputID)
					}
				}
			})

			// Check for main heading
			headings := doc.Find("h1, h2")
			if headings.Length() == 0 {
				t.Error("Template should have at least one main heading")
			}

			// Check for proper form structure
			forms := doc.Find("form")
			if forms.Length() == 0 {
				t.Error("Template should contain at least one form")
			}
		})
	}
}
