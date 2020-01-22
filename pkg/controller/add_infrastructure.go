package controller

import (
	"github.com/dodas-ts/dodas-operator/pkg/controller/infrastructure"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, infrastructure.Add)
}
