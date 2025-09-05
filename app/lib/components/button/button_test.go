
package button

import (
	"context"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestButton(t *testing.T) {
	t.Run("it renders a button element", func(t *testing.T) {
		// Render the component
		var sb strings.Builder
		Button(Props{
			ID:   "my-button",
			Type: TypeButton,
		}).Render(context.Background(), &sb)

		// Parse the rendered HTML
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("failed to read rendered HTML: %v", err)
		}

		// Check that a button element was rendered
		if doc.Find("button").Length() == 0 {
			t.Error("expected a button element to be rendered, but it wasn't")
		}

		// Check the button's attributes
		doc.Find("button").Each(func(i int, s *goquery.Selection) {
			if id, _ := s.Attr("id"); id != "my-button" {
				t.Errorf(`expected id to be "my-button", but got "%s"`, id)
			}
			if typeAttr, _ := s.Attr("type"); typeAttr != "button" {
				t.Errorf(`expected type to be "button", but got "%s"`, typeAttr)
			}
		})
	})

	t.Run("it renders an anchor element when href is provided", func(t *testing.T) {
		// Render the component
		var sb strings.Builder
		Button(Props{
			ID:   "my-link-button",
			Href: "https://example.com",
		}).Render(context.Background(), &sb)

		// Parse the rendered HTML
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("failed to read rendered HTML: %v", err)
		}

		// Check that an anchor element was rendered
		if doc.Find("a").Length() == 0 {
			t.Error("expected an anchor element to be rendered, but it wasn't")
		}

		// Check the anchor's attributes
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			if id, _ := s.Attr("id"); id != "my-link-button" {
				t.Errorf(`expected id to be "my-link-button", but got "%s"`, id)
			}
			if href, _ := s.Attr("href"); href != "https://example.com" {
				t.Errorf(`expected href to be "https://example.com", but got "%s"`, href)
			}
		})
	})

	t.Run("it applies variant and size classes", func(t *testing.T) {
		// Render the component
		var sb strings.Builder
		Button(Props{
			Variant: VariantDestructive,
			Size:    SizeLg,
		}).Render(context.Background(), &sb)

		// Parse the rendered HTML
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
		if err != nil {
			t.Fatalf("failed to read rendered HTML: %v", err)
		}

		// Check that the correct classes are applied
		expectedClasses := []string{"bg-destructive", "h-10"}
		doc.Find("button").Each(func(i int, s *goquery.Selection) {
			for _, class := range expectedClasses {
				if !s.HasClass(class) {
					// Note: This is a simplified check. In a real scenario,
					// you'd need a more robust way to check for Tailwind classes
					// especially with TwMerge. For this example, we'll check substrings.
					classAttr, _ := s.Attr("class")
					if !strings.Contains(classAttr, "bg-destructive") {
						t.Errorf("expected class 'bg-destructive' to be present")
					}
					if !strings.Contains(classAttr, "h-10") {
						t.Errorf("expected class 'h-10' to be present")
					}
				}
			}
		})
	})
}

func TestButtonChildren(t *testing.T) {
	// child := templ.Raw("<span>Click me</span>")

	var sb strings.Builder
	err := Button().Render(context.Background(), &sb)
	if err != nil {
		t.Fatalf("failed to render button: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(sb.String()))
	if err != nil {
		t.Fatalf("failed to read rendered HTML: %v", err)
	}

	if doc.Find("button").Text() != "" {
		t.Errorf("expected button to have no child content, but it did")
	}
}
