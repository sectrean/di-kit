package di_test

import (
	"testing"

	"github.com/sectrean/di-kit"
	"github.com/stretchr/testify/assert"
)

func Test_Lifetime_String(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		lifetime di.Lifetime
	}{
		{
			name:     "singleton",
			lifetime: di.SingletonLifetime,
			want:     "Singleton",
		},
		{
			name:     "transient",
			lifetime: di.TransientLifetime,
			want:     "Transient",
		},
		{
			name:     "scoped",
			lifetime: di.ScopedLifetime,
			want:     "Scoped",
		},
		{
			name:     "unknown lifetime",
			lifetime: di.Lifetime(99),
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
