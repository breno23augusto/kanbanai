package uid

import (
	"github.com/google/uuid"
)

func New() string {
	id, _ := uuid.NewV7()
	return id.String()
}
