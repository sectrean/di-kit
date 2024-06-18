package di

import (
	"reflect"

	"github.com/johnrutherford/di-kit/internal/errors"
)

// WithKey is used to specify the key associated with a service.
//
// WithKey can be used with:
//   - [WithService]
//   - [Resolve]
//   - [MustResolve]
//   - [Container.Resolve]
//   - [Container.Contains]
func WithKey(key any) ServiceKeyOption {
	return keyOption{key}
}

// WithKeyed is used to specify a key for a service dependency when calling
// [WithService] or [Invoke].
//
// This option can be used multiple times to specify keys for function service dependencies.
//
// Example:
//
//	c, err := di.NewContainer(
//		di.WithService(db.NewPrimaryDB, di.WithKey(db.Primary)),
//		di.WithService(db.NewReplicaDB, di.WithKey(db.Replica)),
//		di.WithService(storage.NewReadWriteStore,
//			di.WithKeyed[*db.DB](db.Primary),
//		),
//		di.WithService(storage.NewReadOnlyStore,
//			di.WithKeyed[*db.DB](db.Replica),
//		),
//	)
//
// This option will return an error if the Service does not have a dependency of type Dependency.
func WithKeyed[Dependency any](key any) DependencyKeyOption {
	return depKeyOption{
		t:   reflect.TypeFor[Dependency](),
		key: key,
	}
}

// ServiceKeyOption is used to specify the key associated with a service when calling [WithService],
// [Resolve], [Container.Resolve], or [Container.Contains].
type ServiceKeyOption interface {
	RegisterOption
	ServiceOption
}

// DependencyKeyOption is used to specify a key for a dependency when calling [WithService] or [Invoke].
type DependencyKeyOption interface {
	RegisterOption
	InvokeOption
}

type keyOption struct {
	key any
}

func (o keyOption) applyService(s service) error {
	s.setKey(o.key)
	return nil
}

func (o keyOption) applyServiceKey(key serviceKey) serviceKey {
	return serviceKey{
		Type: key.Type,
		Key:  o.key,
	}
}

var _ ServiceKeyOption = keyOption{}

type depKeyOption struct {
	t   reflect.Type
	key any
}

func (o depKeyOption) applyDeps(deps []serviceKey) error {
	for i := 0; i < len(deps); i++ {
		// Find a dependency with the right type
		// Skip past any that have already been assigned a key
		if deps[i].Type == o.t && deps[i].Key == nil {
			deps[i].Key = o.key
			return nil
		}
	}
	return errors.Errorf("with keyed %s: argument not found", o.t)
}

func (o depKeyOption) applyInvokeConfig(c *invokeConfig) error {
	return o.applyDeps(c.deps)
}

func (o depKeyOption) applyService(s service) error {
	return o.applyDeps(s.Dependencies())
}

var _ DependencyKeyOption = depKeyOption{}
