package di

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/johnrutherford/di-kit/internal/mocks"
	"github.com/johnrutherford/di-kit/internal/testtypes"
)

var (
	InterfaceAType = reflect.TypeFor[testtypes.InterfaceA]()
	InterfaceBType = reflect.TypeFor[testtypes.InterfaceB]()

	InterfaceAKey = serviceKey{Type: InterfaceAType}
	InterfaceBKey = serviceKey{Type: InterfaceBType}
)

func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}

func NewCanceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

func NewDeadlineExceededContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), -1)
	cancel()

	return ctx
}

type testContextKey struct{}

func NewContextWithValue(s string) context.Context {
	return context.WithValue(context.Background(), testContextKey{}, s)
}

func Test_NewContainer(t *testing.T) {
	t.Parallel()

	parent, err := NewContainer(
		WithService(testtypes.NewStructAPtr()),
	)
	assert.NoError(t, err)

	closedParent, err := NewContainer(
		WithService(testtypes.NewStructAPtr()),
	)
	assert.NoError(t, err)

	err = closedParent.Close(context.Background())
	assert.NoError(t, err)

	tests := []struct {
		name    string
		opts    []ContainerOption
		want    *Container
		wantErr string
	}{
		{
			name: "no options",
			opts: nil,
			want: newTestContainer(testContainerConfig{}),
		},
		{
			name: "with parent",
			opts: []ContainerOption{
				WithParent(parent),
			},
			want: newTestContainer(testContainerConfig{
				parent: parent,
			}),
		},
		{
			name: "with closed parent",
			opts: []ContainerOption{
				WithParent(closedParent),
			},
			wantErr: "new container: with parent: container closed",
		},
		{
			name: "with unsupported service kind",
			opts: []ContainerOption{
				WithService(1234),
			},
			wantErr: "new container: with service int: unsupported kind int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewContainer(tt.opts...)
			assert.Equal(t, tt.want, got)

			logError(t, err)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type testContainerConfig struct {
	parent   *Container
	services map[serviceKey]service
	resolved map[serviceKey]resolvedService
	closers  []Closer
	closed   bool
}

func newTestContainer(config testContainerConfig) *Container {
	if config.services == nil {
		config.services = map[serviceKey]service{}
	}
	if config.resolved == nil {
		config.resolved = map[serviceKey]resolvedService{}
	}

	c := &Container{
		parent:   config.parent,
		services: config.services,
		resolved: config.resolved,
		closers:  config.closers,
	}
	c.closed.Store(config.closed)

	return c
}

func Test_Container_Contains(t *testing.T) {
	t.Parallel()

	type args struct {
		t    reflect.Type
		opts []ContainsOption
	}

	tests := []struct {
		name   string
		config testContainerConfig
		args   args
		want   bool
	}{
		{
			name: "type registered",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
			},
			want: true,
		},
		{
			name: "type not registered",
			config: testContainerConfig{
				services: map[serviceKey]service{},
			},
			args: args{
				t: InterfaceAType,
			},
			want: false,
		},
		{
			name: "type registered with tag",
			config: testContainerConfig{
				services: map[serviceKey]service{
					{Type: InterfaceAType, Tag: "tag"}: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
				opts: []ContainsOption{
					WithTag("tag"),
				},
			},
			want: true,
		},
		{
			name: "type not registered with tag",
			config: testContainerConfig{
				services: map[serviceKey]service{
					{Type: InterfaceAType, Tag: "tag"}: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
				opts: []ContainsOption{
					WithTag("other"),
				},
			},
			want: false,
		},
		{
			name: "type registered in parent",
			config: testContainerConfig{
				parent: &Container{
					services: map[serviceKey]service{
						InterfaceAKey: &funcService{},
					},
				},
			},
			args: args{
				t: InterfaceAType,
			},
			want: true,
		},
		{
			name: "type not registered in parent",
			config: testContainerConfig{
				parent: &Container{
					services: map[serviceKey]service{},
				},
			},
			args: args{
				t: InterfaceAType,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(tt.config)

			got := c.Contains(tt.args.t, tt.args.opts...)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Container_Resolve(t *testing.T) {
	t.Parallel()

	ctxWithValue := NewContextWithValue("test")

	type args struct {
		ctx context.Context
		t   reflect.Type
	}

	tests := []struct {
		name      string
		config    testContainerConfig
		args      args
		want      any
		wantSame  bool
		wantErr   string
		wantErrIs error
	}{
		{
			name: "container closed",
			config: testContainerConfig{
				closed: true,
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			want:      nil,
			wantErr:   "resolve testtypes.InterfaceA: container closed",
			wantErrIs: ErrContainerClosed,
		},
		{
			name: "context canceled",
			args: args{
				ctx: NewCanceledContext(),
				t:   reflect.TypeFor[testtypes.InterfaceA](),
			},
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(testtypes.NewInterfaceA)),
				},
			},
			want:      nil,
			wantErr:   "resolve testtypes.InterfaceA: context canceled",
			wantErrIs: context.Canceled,
		},
		{
			name: "context deadline exceeded",
			args: args{
				ctx: NewDeadlineExceededContext(),
				t:   reflect.TypeFor[testtypes.InterfaceA](),
			},
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(testtypes.NewInterfaceA)),
				},
			},
			want:      nil,
			wantErr:   "resolve testtypes.InterfaceA: context deadline exceeded",
			wantErrIs: context.DeadlineExceeded,
		},
		{
			name: "resolve context.Context",
			args: args{
				ctx: ctxWithValue,
				t:   reflect.TypeFor[context.Context](),
			},
			want:     ctxWithValue,
			wantSame: true,
		},
		{
			name: "dependency cycle",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(testtypes.NewInterfaceADependsOnB)),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceBDependsOnA)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			wantErr: "resolve testtypes.InterfaceA: resolve dependency testtypes.InterfaceB: " +
				"resolve dependency testtypes.InterfaceA: dependency cycle detected",
			wantErrIs: ErrDependencyCycle,
		},
		{
			name: "type not registered",
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			wantErr:   "resolve testtypes.InterfaceA: type not registered",
			wantErrIs: ErrTypeNotRegistered,
		},
		{
			name: "one service no dependencies",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(testtypes.NewInterfaceA)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			want: testtypes.NewInterfaceA(),
		},
		{
			name: "one interface value",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newValueService(testtypes.NewInterfaceA())),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			want: testtypes.NewInterfaceA(),
		},
		{
			name: "one dependency",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(testtypes.NewInterfaceA)),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceBDependsOnA)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceB](),
			},
			want: testtypes.NewInterfaceB(),
		},
		{
			name: "one value dependency",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newValueService(testtypes.NewInterfaceA())),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceBDependsOnA)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceB](),
			},
			want: testtypes.NewInterfaceB(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(tt.config)

			// Default ctx arg
			if tt.args.ctx == nil {
				tt.args.ctx = context.Background()
			}

			got, err := c.Resolve(tt.args.ctx, tt.args.t)

			if tt.wantSame {
				assert.Same(t, tt.want, got)
			} else {
				assert.Equal(t, tt.want, got)
			}

			logError(t, err)

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

func Test_Container_Close(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  testContainerConfig
		ctx     context.Context
		wantErr string
	}{
		{
			name: "already closed",
			config: testContainerConfig{
				closed: true,
			},
			ctx:     context.Background(),
			wantErr: "already closed: container closed",
		},
		{
			name: "no closers",
			config: testContainerConfig{
				closers: []Closer{},
			},
			ctx: context.Background(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(tt.config)

			// Default ctx arg
			if tt.ctx == nil {
				tt.ctx = context.Background()
			}

			// Call the Close method
			err := c.Close(tt.ctx)
			logError(t, err)

			// Check the error
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Container_CloseWithClosers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		configFunc func(t *testing.T) testContainerConfig
		ctx        context.Context
		wantErr    string
	}{
		{
			name: "single closer",
			configFunc: func(t *testing.T) testContainerConfig {
				aMock := mocks.NewInterfaceAMock(t)
				aMock.EXPECT().
					Close(mock.Anything).
					Return(nil).
					Once()

				return testContainerConfig{
					closers: []Closer{aMock},
				}
			},
			ctx: context.Background(),
		},
		{
			name: "single close error",
			configFunc: func(t *testing.T) testContainerConfig {
				aMock := mocks.NewInterfaceAMock(t)
				aMock.EXPECT().
					Close(mock.Anything).
					Return(errors.New("mocked error")).
					Once()

				return testContainerConfig{
					closers: []Closer{aMock},
				}
			},
			ctx:     context.Background(),
			wantErr: "close container: mocked error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc(t)
			c := newTestContainer(config)

			// Default ctx arg
			if tt.ctx == nil {
				tt.ctx = context.Background()
			}

			// Call the Close method
			err := c.Close(tt.ctx)
			logError(t, err)

			// Check the error
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func logError(t *testing.T, err error) {
	if err == nil {
		return
	}

	t.Helper()
	t.Logf("error message:\n%v", err)
}
