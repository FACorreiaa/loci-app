
package auth

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestSignInPage(t *testing.T) {
	t.Run("it renders the sign-in form", func(t *testing.T) {
		// Render the component
		var sb strings.Builder
		err := SignIn().Render(context.Background(), &sb)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		// Parse the rendered HTML
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("failed to read rendered HTML: %v", err)
		}

		// Check for the form
		form := doc.Find("form")
		if form.Length() == 0 {
			t.Error("expected a form element to be rendered, but it wasn't")
		}

		// Check for hx-post attribute
		if hxPost, _ := form.Attr("hx-post"); hxPost != "/auth/signin" {
			t.Errorf(`expected hx-post attribute to be "/auth/signin", but got "%s"`, hxPost)
		}

		// Check for email input
		emailInput := form.Find("input[name='email']")
		if emailInput.Length() == 0 {
			t.Error("expected an email input element to be rendered, but it wasn't")
		}

		// Check for password input
		passwordInput := form.Find("input[name='password']")
		if passwordInput.Length() == 0 {
			t.Error("expected a password input element to be rendered, but it wasn't")
		}

		// Check for submit button
		submitButton := form.Find("button[type='submit']")
		if submitButton.Length() == 0 {
			t.Error("expected a submit button to be rendered, but it wasn't")
		}
	})

	t.Run("it has a link to the sign-up page", func(t *testing.T) {
		// Render the component
		var sb strings.Builder
		err := SignIn().Render(context.Background(), &sb)
		if err != nil {
			t.Fatalf("failed to render: %v", err)
		}

		// Parse the rendered HTML
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("failed to read rendered HTML: %v", err)
		}

		// Check for the sign-up link
		signUpLink := doc.Find("a[href='/auth/signup']")
		if signUpLink.Length() == 0 {
			t.Error("expected a link to the sign-up page, but it wasn't found")
		}
	})
}
