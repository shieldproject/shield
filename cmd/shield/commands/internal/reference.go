package internal

import (
	"fmt"

	"github.com/pborman/uuid"
)

type Reference struct {
	UUID uuid.UUID
	Name string
}

func (r *Reference) HumanReadable() string {
	return fmt.Sprintf("%s (%s)", r.Name, r.UUID.String())
}

func (r *Reference) MachineReadable() interface{} {
	return r.UUID
}

func NewReference(id string, name string) *Reference {
	return &Reference{
		UUID: uuid.Parse(id),
		Name: name,
	}
}
