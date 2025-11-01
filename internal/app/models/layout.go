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
