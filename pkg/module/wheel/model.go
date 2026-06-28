package wheel

import (
	"time"

	"github.com/mymmrac/wof/pkg/module/id"
)

type Model struct {
	ID        id.ID     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
