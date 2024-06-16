package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Lifetime_String(t *testing.T) {
	tests := []struct {
		name     string
		lifetime Lifetime
		want     string
	}{
		{
			name:     "singleton",
			lifetime: Singleton,
			want:     "Singleton",
		},
		{
			name:     "transient",
			lifetime: Transient,
			want:     "Transient",
		},
		{
			name:     "scoped",
			lifetime: Scoped,
			want:     "Scoped",
		},
		{
			name:     "unknown lifetime",
			lifetime: Lifetime(99),
			want:     "Unknown Lifetime 99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.lifetime.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
