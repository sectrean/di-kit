package di

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InterfaceA interface {
	A()
}

type InterfaceB interface {
	B()
}

type StructA struct{}

func (StructA) A() {}

type StructB struct{}

func (StructB) B() {}

type testContextKey struct{}

var (
	ctxCanceled         context.Context
	ctxDeadlineExceeded context.Context
	ctxWithValue        context.Context
)

func init() {
	var cancel context.CancelFunc
	ctxCanceled, cancel = context.WithCancel(context.Background())
	cancel()
	ctxDeadlineExceeded, cancel = context.WithTimeout(context.Background(), -1)
	cancel()
	ctxWithValue = context.WithValue(context.Background(), testContextKey{}, "test")
}

// func newAtomicBool(val bool) *atomic.Bool {
// 	b := new(atomic.Bool)
// 	b.Store(val)

// 	return b
// }

func newA() InterfaceA {
	return &StructA{}
}

func newB() InterfaceB {
	return &StructB{}
}

func newADependsOnB(_ InterfaceB) InterfaceA {
	return &StructA{}
}

func newBDependsOnA(_ InterfaceA) InterfaceB {
	return &StructB{}
}

// TODO: Test constructors functions with errors

func Test_Container_Resolve(t *testing.T) {
	type fields struct {
		parent    *Container
		services  map[reflect.Type]Service
		resolved  map[reflect.Type]resolvedService
		resolveMu *sync.Mutex
		closers   []Closer
		closed    *atomic.Bool
	}
	type args struct {
		ctx context.Context
		typ reflect.Type
	}

	tests := []struct {
		name      string
		fields    fields
		args      args
		want      any
		wantSame  bool
		wantErr   string
		wantErrIs error
	}{
		// {
		// 	name: "container closed",
		// 	fields: fields{
		// 		closed: newAtomicBool(true),
		// 	},
		// 	args: args{
		// 		typ: TypeOf[InterfaceA](),
		// 	},
		// 	want:      nil,
		// 	wantErr:   "resolving type di.InterfaceA: container closed",
		// 	wantErrIs: ErrContainerClosed,
		// },
		{
			name: "context canceled",
			args: args{
				ctx: ctxCanceled,
				typ: TypeOf[InterfaceA](),
			},
			want:      nil,
			wantErr:   "resolving type di.InterfaceA: context canceled",
			wantErrIs: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			args: args{
				ctx: ctxDeadlineExceeded,
				typ: TypeOf[InterfaceA](),
			},
			want:      nil,
			wantErr:   "resolving type di.InterfaceA: context deadline exceeded",
			wantErrIs: context.DeadlineExceeded,
		},
		{
			name: "resolve context.Context",
			args: args{
				ctx: ctxWithValue,
				typ: TypeOf[context.Context](),
			},
			want:     ctxWithValue,
			wantSame: true,
		},
		{
			name: "dependency cycle",
			fields: fields{
				services: map[reflect.Type]Service{
					TypeOf[InterfaceA](): Must(NewService(newADependsOnB)),
					TypeOf[InterfaceB](): Must(NewService(newBDependsOnA)),
				},
			},
			args: args{
				typ: TypeOf[InterfaceA](),
			},
			wantErr: "resolving type di.InterfaceA: resolving dependency di.InterfaceB: " +
				"resolving dependency di.InterfaceA: dependency cycle detected",
			wantErrIs: ErrDependencyCycle,
		},
		{
			name: "type not registered",
			args: args{
				typ: TypeOf[InterfaceA](),
			},
			wantErr:   "resolving type di.InterfaceA: type not registered",
			wantErrIs: ErrTypeNotRegistered,
		},
		{
			name: "one service no dependencies",
			fields: fields{
				services: map[reflect.Type]Service{
					TypeOf[InterfaceA](): Must(NewService(newA)),
				},
			},
			args: args{
				typ: TypeOf[InterfaceA](),
			},
			want: newA(),
		},
		{
			name: "one interface value",
			fields: fields{
				services: map[reflect.Type]Service{
					TypeOf[InterfaceA](): Must(NewService(newA())),
				},
			},
			args: args{
				typ: TypeOf[InterfaceA](),
			},
			want: newA(),
		},
		{
			name: "one dependency",
			fields: fields{
				services: map[reflect.Type]Service{
					TypeOf[InterfaceA](): Must(NewService(newA)),
					TypeOf[InterfaceB](): Must(NewService(newBDependsOnA)),
				},
			},
			args: args{
				typ: TypeOf[InterfaceB](),
			},
			want: newB(),
		},
		{
			name: "one value dependency",
			fields: fields{
				services: map[reflect.Type]Service{
					TypeOf[InterfaceA](): Must(NewService(newA())),
					TypeOf[InterfaceB](): Must(NewService(newBDependsOnA)),
				},
			},
			args: args{
				typ: TypeOf[InterfaceB](),
			},
			want: newB(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Default field values
			if tt.fields.services == nil {
				tt.fields.services = make(map[reflect.Type]Service)
			}
			if tt.fields.resolved == nil {
				tt.fields.resolved = make(map[reflect.Type]resolvedService)
			}
			if tt.fields.resolveMu == nil {
				tt.fields.resolveMu = new(sync.Mutex)
			}
			if tt.fields.closed == nil {
				tt.fields.closed = new(atomic.Bool)
			}

			// Set up the container
			c := &Container{
				parent:    tt.fields.parent,
				services:  tt.fields.services,
				resolved:  tt.fields.resolved,
				resolveMu: tt.fields.resolveMu,
				closers:   tt.fields.closers,
				closed:    tt.fields.closed,
			}

			// Default ctx argument
			if tt.args.ctx == nil {
				tt.args.ctx = context.Background()
			}

			got, err := c.Resolve(tt.args.ctx, tt.args.typ)

			if tt.wantSame {
				assert.Same(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want, got)
			}

			if err != nil {
				t.Logf("error message:\n%v", err)
			}

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			}
			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
			}
			if tt.wantErr == "" && tt.wantErrIs == nil {
				assert.NoError(t, err)
			}
		})
	}
}
