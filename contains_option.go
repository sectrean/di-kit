package di

// ContainsOption is a functional option for [Scope.Contains].
//
// Available options:
//   - [WithTag] specifies the tag associated with a service.
type ContainsOption interface {
	applyContainsKey(serviceKey) serviceKey
}
