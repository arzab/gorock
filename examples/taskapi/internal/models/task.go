package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Task struct {
	bun.BaseModel `bun:"table:tasks" json:"-"`

	ID        string    `bun:"id,pk" json:"id"`
	Title     string    `bun:"title,notnull" json:"title"`
	Done      bool      `bun:"done,notnull" json:"done"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
}
