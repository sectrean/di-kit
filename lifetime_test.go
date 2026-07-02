package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Lifetime_string(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		lifetime Lifetime
	}{
		{
			name:     "singleton",
			lifetime: Singleton,
			want:     "di.Singleton",
		},
		{
			name:     "transient",
			lifetime: Transient,
			want:     "di.Transient",
		},
		{
			name:     "scoped",
			lifetime: Scoped,
			want:     "di.Scoped",
		},
		{
			name:     "unknown",
			lifetime: Lifetime(99),
			want:     "di.Lifetime(99)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lifetime.string()
			assert.Equal(t, tt.want, got)
		})
	}
}
