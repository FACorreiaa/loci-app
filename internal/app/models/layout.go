package models

import "github.com/a-h/templ"

type User struct {
	ID       string
	Name     string
	Email    string
	IsActive bool
}

type NavItem struct {
	Name string
	URL  string
	Icon string
}

type Navigation struct {
	Items []NavItem
}

type LayoutTempl struct {
	Title     string
	User      *User
	Nav       Navigation
	ActiveNav string
	Content   templ.Component
}

var MainNav = Navigation{
	Items: []NavItem{
		{Name: "Dashboard", URL: "/dashboard"},
		{Name: "Discover", URL: "/discover"},
		{Name: "Nearby", URL: "/nearby"},
		{Name: "Chat", URL: "/chat"},
		{Name: "Favorites", URL: "/favorites"},
	},
}

var OfflineNav = Navigation{
	Items: []NavItem{
		{Name: "About", URL: "/about"},
		{Name: "Features", URL: "/features"},
		{Name: "Pricing", URL: "/pricing"},
	},
}
