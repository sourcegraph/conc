package stream

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleStream() {
	s := New()
	for i := 0; i < 5; i++ {
		i := i
		s.Go(func() Callback {
			i *= 2
			return func() { fmt.Println(i) }
		})
	}
	s.Wait()
	// Output:
	// 0
	// 2
	// 4
	// 6
	// 8
}

func TestStream(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		s := New()
		var res []int
		for i := 0; i < 5; i++ {
			i := i
			s.Go(func() Callback {
				i *= 2
				return func() {
					res = append(res, i)
				}
			})
		}
		s.Wait()
		require.Equal(t, []int{0, 2, 4, 6, 8}, res)
	})

}
