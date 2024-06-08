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

func ContextCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

func ContextDeadlineExceeded() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), -1)
	cancel()

	return ctx
}

type testContextKey struct{}

func ContextWithValue(s string) context.Context {
	return context.WithValue(context.Background(), testContextKey{}, s)
}

func Test_NewContainer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []ContainerOption
		want    func(*testing.T, *Container)
		wantErr string
	}{
		{
			name: "no options",
			opts: nil,
			want: func(t *testing.T, c *Container) {
				assert.NotNil(t, c)
				assert.Len(t, c.services, 0)
			},
		},
		{
			name: "with unsupported service kind",
			opts: []ContainerOption{
				Register(1234),
			},
			wantErr: "new container: register int: unsupported kind int",
		},
		{
			name: "with nil service",
			opts: []ContainerOption{
				Register((testtypes.InterfaceA)(nil)),
			},
			wantErr: "new container: register: funcOrValue is nil",
		},
		{
			name: "no service, only options",
			opts: []ContainerOption{
				Register(Singleton, WithKey("key")),
			},
			wantErr: "new container: register di.Lifetime: unexpected RegisterOption as funcOrValue",
		},
		{
			name: "WithKeyed with dep not found",
			opts: []ContainerOption{
				Register(testtypes.NewInterfaceA, WithKeyed[testtypes.InterfaceB]("key")),
			},
			wantErr: "new container: register func() testtypes.InterfaceA: with keyed testtypes.InterfaceB: argument not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewContainer(tt.opts...)
			if tt.want != nil {
				tt.want(t, got)
			}

			logErrorMessage(t, err)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Container_NewScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parent  testContainerConfig
		opts    []ContainerOption
		want    func(*testing.T, *Container)
		wantErr string
	}{
		{
			name: "success",
			parent: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: &funcService{},
				},
			},
			want: func(t *testing.T, c *Container) {
				assert.NotNil(t, c)
				assert.NotNil(t, c.parent)
				assert.Equal(t, c.parent.services, c.services)
			},
		},
		{
			name: "success with register",
			parent: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: &funcService{},
				},
			},
			opts: []ContainerOption{
				Register(testtypes.NewInterfaceB),
			},
			want: func(t *testing.T, c *Container) {
				assert.NotNil(t, c)
				assert.NotNil(t, c.parent)
				assert.NotEqual(t, c.parent.services, c.services)

				assert.Len(t, c.services, 2)
				assert.Contains(t, c.services, InterfaceAKey)
				assert.Contains(t, c.services, InterfaceBKey)
			},
		},
		{
			name: "parent container closed",
			parent: testContainerConfig{
				closed: true,
			},
			want: func(t *testing.T, c *Container) {
				assert.Nil(t, c)
			},
			wantErr: "new scope: container closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(t, tt.parent)
			scope, err := c.NewScope(tt.opts...)

			logErrorMessage(t, err)

			if tt.want != nil {
				tt.want(t, scope)
			}

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				assert.Nil(t, scope)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scope)
			}
		})
	}
}

type testContainerConfig struct {
	parent   *Container
	services map[serviceKey]service
	resolved map[serviceKey]*servicePromise
	closers  []Closer
	closed   bool
	setup    func(t *testing.T, c *testContainerConfig)
}

func newTestContainer(t *testing.T, config testContainerConfig) *Container {
	if config.services == nil {
		config.services = map[serviceKey]service{}
	}
	if config.resolved == nil {
		config.resolved = map[serviceKey]*servicePromise{}
	}

	if config.setup != nil {
		config.setup(t, &config)
	}

	resolved := make(map[service]*servicePromise, len(config.resolved))
	for k, v := range config.resolved {
		svc := config.services[k]
		resolved[svc] = v
	}

	c := &Container{
		parent:   config.parent,
		services: config.services,
		resolved: resolved,
		closers:  config.closers,
		closed:   config.closed,
	}

	return c
}

func Test_Container_Contains(t *testing.T) {
	t.Parallel()

	type args struct {
		t    reflect.Type
		opts []ServiceOption
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
			name: "type registered with key",
			config: testContainerConfig{
				services: map[serviceKey]service{
					{Type: InterfaceAType, Key: "key"}: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
				opts: []ServiceOption{
					WithKey("key"),
				},
			},
			want: true,
		},
		{
			name: "type not registered with key",
			config: testContainerConfig{
				services: map[serviceKey]service{
					{Type: InterfaceAType, Key: "key"}: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
				opts: []ServiceOption{
					WithKey("other"),
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
			c := newTestContainer(t, tt.config)

			got := c.Contains(tt.args.t, tt.args.opts...)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Container_Resolve(t *testing.T) {
	t.Parallel()

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
				ctx: ContextCanceled(),
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
				ctx: ContextDeadlineExceeded(),
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
				ctx: context.Background(),
				t:   reflect.TypeFor[context.Context](),
			},
			wantErr: "resolve context.Context: type not registered",
		},
		{
			name: "dependency cycle",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newFuncService(func(testtypes.InterfaceB) testtypes.InterfaceA { return nil })),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceB)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceA](),
			},
			wantErr: "resolve testtypes.InterfaceA: dependency testtypes.InterfaceB: " +
				"dependency testtypes.InterfaceA: dependency cycle detected",
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
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceB)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceB](),
			},
			want: &testtypes.StructB{},
		},
		{
			name: "one value dependency",
			config: testContainerConfig{
				services: map[serviceKey]service{
					InterfaceAKey: Must(newValueService(testtypes.NewInterfaceA())),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceB)),
				},
			},
			args: args{
				t: reflect.TypeFor[testtypes.InterfaceB](),
			},
			want: &testtypes.StructB{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(t, tt.config)

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

			logErrorMessage(t, err)

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
			wantErr: "close: already closed: container closed",
		},
		{
			name:   "no closers",
			config: testContainerConfig{},
			ctx:    context.Background(),
		},
		{
			name: "single closer",
			config: testContainerConfig{
				setup: func(t *testing.T, c *testContainerConfig) {
					aMock := mocks.NewInterfaceAMock(t)
					aMock.EXPECT().
						Close(mock.Anything).
						Return(nil).
						Once()

					c.closers = []Closer{aMock}
				},
			},
			ctx: context.Background(),
		},
		{
			name: "single close error",
			config: testContainerConfig{
				setup: func(t *testing.T, c *testContainerConfig) {
					aMock := mocks.NewInterfaceAMock(t)
					aMock.EXPECT().
						Close(mock.Anything).
						Return(errors.New("mocked error")).
						Once()

					c.closers = []Closer{aMock}
				},
			},
			ctx:     context.Background(),
			wantErr: "close container: mocked error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestContainer(t, tt.config)

			// Call the Close method
			err := c.Close(tt.ctx)
			logErrorMessage(t, err)

			// Check the error
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func logErrorMessage(t *testing.T, err error) {
	if err == nil {
		return
	}

	// We log our error messages so we can make sure they are helpful and informative
	t.Helper()
	t.Logf("error message:\n%v", err)
}
