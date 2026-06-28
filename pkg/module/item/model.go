package item

import (
	"time"

	"github.com/mymmrac/wof/pkg/module/id"
)

type Model struct {
	ID        id.ID     `json:"id"`
	WheelID   id.ID     `json:"wheelId"`
	Name      string    `json:"name,omitempty"`
	Order     int       `json:"order"`
	Rating    int       `json:"rating,omitempty"`
	Rejected  bool      `json:"rejected,omitempty"`
	Used      bool      `json:"used,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
