package di

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/puzpuzpuz/xsync/v3"
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

	parent, err := NewContainer(
		Register(testtypes.NewStructAPtr()),
	)
	assert.NoError(t, err)

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
			name: "with parent",
			opts: []ContainerOption{
				WithParent(parent),
			},
			want: func(t *testing.T, c *Container) {
				assert.NotNil(t, c)
				assert.Same(t, parent, c.parent)
			},
		},
		{
			name: "with closed parent",
			opts: []ContainerOption{
				WithParent(newTestContainer(t,
					testContainerConfig{
						closed: true,
					},
				)),
			},
			wantErr: "new container: with parent: container closed",
		},
		{
			name: "with unsupported service kind",
			opts: []ContainerOption{
				Register(1234),
			},
			wantErr: "new container: with service int: unsupported kind int",
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

type testContainerConfig struct {
	parent   *Container
	services map[serviceKey]service
	resolved map[serviceKey]*resolveFuture
	closers  []Closer
	closed   bool
	setup    func(t *testing.T, c *testContainerConfig)
}

func newTestContainer(t *testing.T, config testContainerConfig) *Container {
	if config.services == nil {
		config.services = map[serviceKey]service{}
	}
	if config.resolved == nil {
		config.resolved = map[serviceKey]*resolveFuture{}
	}

	if config.setup != nil {
		config.setup(t, &config)
	}

	resolved := xsync.NewMapOfPresized[serviceKey, *resolveFuture](len(config.resolved))
	for k, v := range config.resolved {
		resolved.Store(k, v)
	}

	c := &Container{
		parent:   config.parent,
		services: config.services,
		resolved: resolved,
		closers:  config.closers,
		closeMu:  xsync.NewRBMutex(),
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
			name: "type registered with tag",
			config: testContainerConfig{
				services: map[serviceKey]service{
					{Type: InterfaceAType, Tag: "tag"}: &funcService{},
				},
			},
			args: args{
				t: InterfaceAType,
				opts: []ServiceOption{
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
				opts: []ServiceOption{
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
			c := newTestContainer(t, tt.config)

			got := c.Contains(tt.args.t, tt.args.opts...)

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_Container_Resolve(t *testing.T) {
	t.Parallel()

	ctxWithValue := ContextWithValue("test")

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
					InterfaceAKey: Must(newFuncService(func(testtypes.InterfaceB) testtypes.InterfaceA { return nil })),
					InterfaceBKey: Must(newFuncService(testtypes.NewInterfaceB)),
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
			wantErr: "already closed: container closed",
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

	t.Helper()
	t.Logf("error message:\n%v", err)
}
