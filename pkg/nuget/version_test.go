package nuget_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khulnasoft/go-version-lib/pkg/nuget"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.2.3", "1.2.3", 0},
		{"1.2.3", "1.2.4", -1},
		{"1.2.3.4", "1.2.3", 1},
		{"1.2", "1.2.0.0", 0},
		{"1.2.3-alpha", "1.2.3", -1},
		{"1.2.3", "1.2.3-alpha", 1},
		{"1.2.3-alpha", "1.2.3-alpha.1", -1},
		{"1.2.3-alpha.1", "1.2.3-alpha.beta", -1},
		{"1.2.3-1", "1.2.3-alpha", -1},
	}
	for _, tc := range cases {
		t.Run(tc.a+" vs "+tc.b, func(t *testing.T) {
			va, err := nuget.NewVersion(tc.a)
			require.NoError(t, err)
			vb, err := nuget.NewVersion(tc.b)
			require.NoError(t, err)
			assert.Equal(t, tc.want, va.Compare(vb))
			assert.Equal(t, -tc.want, vb.Compare(va))
		})
	}
}

func TestInvalid(t *testing.T) {
	for _, s := range []string{"", "foo", "1.2.3.4.5", "1..2"} {
		_, err := nuget.NewVersion(s)
		assert.Error(t, err, s)
	}
}
