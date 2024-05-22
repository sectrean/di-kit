package di

// ResolveOption can be used when calling [Container.Resolve] and [Resolve].
//
// Available options:
//   - [WithTag]
type ResolveOption interface {
	applyResolveKey(serviceKey) serviceKey
}
