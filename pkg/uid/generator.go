package uid

import (
	"github.com/google/uuid"
)

func New() string {
	return uuid.NewV7().String()
}
