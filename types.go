package di

import (
	"context"
)

// Common types used in the package
var (
	typError   = TypeOf[error]()
	typContext = TypeOf[context.Context]()
	typScope   = TypeOf[Scope]()
)
