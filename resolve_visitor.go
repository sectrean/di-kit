package di

import (
	"reflect"
	"strings"

	"github.com/google/uuid"
)

type resolveVisitor struct {
	Scope   *container
	Root    *container
	visited map[uuid.UUID]Service
	trail   []reflect.Type
}

func newResolveVisitor(scope *container) *resolveVisitor {
	return &resolveVisitor{
		Scope:   scope,
		Root:    scope.getRoot(),
		visited: make(map[uuid.UUID]Service),
	}
}

// Enter returns true if the service has already been visited
func (v *resolveVisitor) Enter(svc Service) bool {
	if _, ok := v.visited[svc.ID()]; ok {
		return true
	}
	v.visited[svc.ID()] = svc
	v.trail = append(v.trail, svc.Type())
	return false
}

func (v *resolveVisitor) Leave() {
	v.trail = v.trail[:len(v.trail)-1]
}

// Trail returns a string representation of the dependency chain
func (v *resolveVisitor) Trail(t reflect.Type) string {
	trail := append(v.trail, t)
	types := make([]string, len(trail))
	for i, svc := range trail {
		types[i] = svc.String()
	}
	return strings.Join(types, " -> ")
}
