package model

import "time"

// Kanban represents a kanban board
type Kanban struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Columns     []Column  `json:"columns"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Column represents a column in a kanban board
type Column struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
	Cards []Card `json:"cards"`
}

// Card represents a card in a kanban column
type Card struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Order       int       `json:"order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// KanbanStore holds all kanban boards
type KanbanStore struct {
	Kanbans []Kanban `json:"kanbans"`
}
