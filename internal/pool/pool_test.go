package pool

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bazueva/metrics/internal/pool/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("создание пула", func(t *testing.T) {
		t.Parallel()

		p := New(func() *mocks.MockResetter {
			return mocks.NewMockResetter(t)
		})

		require.NotNil(t, p)
	})
}

func TestPoolGet(t *testing.T) {
	t.Parallel()

	t.Run("создание нового объекта при пустом пуле", func(t *testing.T) {
		t.Parallel()

		var created int

		object := mocks.NewMockResetter(t)

		p := New(func() *mocks.MockResetter {
			created++
			return object
		})

		actual := p.Get()

		assert.Same(t, object, actual)
		assert.Equal(t, 1, created)
	})

	t.Run("повторное использование объекта", func(t *testing.T) {
		t.Parallel()

		var created int

		object := mocks.NewMockResetter(t)
		object.EXPECT().Reset().Return()

		p := New(func() *mocks.MockResetter {
			created++
			return object
		})

		first := p.Get()
		p.Put(first)

		second := p.Get()

		assert.Same(t, first, second)
		assert.Equal(t, 1, created)
	})

	t.Run("извлечение объектов", func(t *testing.T) {
		t.Parallel()

		first := mocks.NewMockResetter(t)
		second := mocks.NewMockResetter(t)

		first.EXPECT().Reset().Return()
		second.EXPECT().Reset().Return()

		p := New(func() *mocks.MockResetter {
			return mocks.NewMockResetter(t)
		})

		p.Put(first)
		p.Put(second)

		assert.Same(t, second, p.Get())
		assert.Same(t, first, p.Get())
	})

	t.Run("создание нового объекта после исчерпания пула", func(t *testing.T) {
		t.Parallel()

		var created int

		p := New(func() *mocks.MockResetter {
			created++
			return mocks.NewMockResetter(t)
		})

		first := p.Get()
		first.EXPECT().Reset().Return()

		p.Put(first)

		reused := p.Get()
		newObject := p.Get()

		assert.Same(t, first, reused)
		assert.NotSame(t, reused, newObject)
		assert.Equal(t, 2, created)
	})
}

func TestPoolPut(t *testing.T) {
	t.Parallel()

	t.Run("вызов Reset при помещении в пул", func(t *testing.T) {
		t.Parallel()

		object := mocks.NewMockResetter(t)
		object.EXPECT().Reset().Return()

		p := New(func() *mocks.MockResetter {
			return mocks.NewMockResetter(t)
		})

		p.Put(object)

		assert.Same(t, object, p.Get())
	})

	t.Run("повторный вызов Reset при каждом Put", func(t *testing.T) {
		t.Parallel()

		object := mocks.NewMockResetter(t)
		object.EXPECT().Reset().Return().Twice()

		p := New(func() *mocks.MockResetter {
			return object
		})

		p.Put(object)
		reused := p.Get()
		p.Put(reused)

		assert.Same(t, object, p.Get())
	})
}

func TestPoolConcurrent(t *testing.T) {
	t.Parallel()

	t.Run("конкурентные Get и Put", func(t *testing.T) {
		t.Parallel()

		var created atomic.Int64

		p := New(func() *mocks.MockResetter {
			created.Add(1)

			object := mocks.NewMockResetter(t)
			object.EXPECT().Reset().Return()

			return object
		})

		const goroutines = 100
		const iterations = 100

		var wg sync.WaitGroup

		for range goroutines {
			wg.Add(1)

			go func() {
				defer wg.Done()

				for range iterations {
					object := p.Get()
					p.Put(object)
				}
			}()
		}

		wg.Wait()

		assert.Greater(t, created.Load(), int64(0))
	})
}
