//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/typestep
//

package core

import "fmt"

type Account struct {
	ID string `json:"id"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetUser(acc Account) (User, error) {
	return User{
		ID:   acc.ID,
		Name: "Alice",
	}, nil
}

// Category that is recommended to a user
type Category struct {
	ID   string `json:"id"`
	User User   `json:"user"`
}

func PickCategory(user User) ([]Category, error) {
	return []Category{
		{ID: "A", User: user},
		{ID: "B", User: user},
		{ID: "C", User: user},
	}, nil
}

// Product that is recommended to a user
type Product struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	User     User   `json:"user"`
}

func PickProduct(cat Category) ([]Product, error) {
	return []Product{
		{ID: "001", Category: cat.ID, User: cat.User},
		{ID: "002", Category: cat.ID, User: cat.User},
		{ID: "003", Category: cat.ID, User: cat.User},
		{ID: "004", Category: cat.ID, User: cat.User},
		{ID: "005", Category: cat.ID, User: cat.User},
	}, nil
}

func MailTo(p Product) (string, error) {
	tmpl := `
Dear %s!

Check out our the %s pick from this category %s—you might find exactly what you need!
`
	return fmt.Sprintf(tmpl, p.User.Name, p.ID, p.Category), nil
}
