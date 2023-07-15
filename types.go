package di

import "context"

var (
	tError     = TypeOf[error]()
	tContext   = TypeOf[context.Context]()
	tContainer = TypeOf[Container]()
)
