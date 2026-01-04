package singleflightx

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGroup_Do(t *testing.T) {
	t.Run("success single call", func(t *testing.T) {
		wantVal := "foo bar"

		var g Group[string, string]
		v, err, shared := g.Do("key", func() (string, error) {
			return wantVal, nil
		})
		require.NoError(t, err)
		require.Equal(t, wantVal, v)
		require.False(t, shared)
	})

	t.Run("success concurrent calls", func(t *testing.T) {
		wantVal := "foo bar"

		var g Group[string, string]

		block := make(chan struct{})
		unblock := make(chan struct{})

		var v1, v2 string
		var err1, err2 error
		var shared1, shared2 bool
		go func() {
			v1, err1, shared1 = g.Do("key", func() (string, error) {
				close(block)
				<-unblock
				return wantVal, nil
			})
		}()

		<-block

		go func() {
			close(unblock)
			v2, err2, shared2 = g.Do("key", func() (string, error) {
				require.FailNow(t, "second call should not be executed")
				return "", nil
			})
		}()

		<-unblock

		require.NoError(t, err1)
		require.Equal(t, wantVal, v1)
		require.True(t, shared1)

		require.NoError(t, err2)
		require.Equal(t, wantVal, v2)
		require.True(t, shared2)
	})

	t.Run("nil", func(t *testing.T) {
		var g Group[string, any]
		v, err, _ := g.Do("key", func() (any, error) {
			return nil, nil
		})

		require.NoError(t, err)
		require.Nil(t, v)
	})

	t.Run("error", func(t *testing.T) {
		wantErr := errors.New("test error")

		var g Group[string, string]
		v, err, _ := g.Do("key", func() (string, error) {
			return "", wantErr
		})

		require.ErrorIs(t, err, wantErr)
		require.Equal(t, "", v)
	})
}

func runParallelDoGrouped(b *testing.B, total int, groupSize int) {
	var g Group[string, int]

	fn := func() (int, error) {
		// имитация работы функции
		time.Sleep(time.Millisecond)
		return 42, nil
	}

	uniqueKeys := total / groupSize

	for i := 0; i < b.N; i++ {
		start := make(chan struct{})
		done := make(chan struct{}, total)

		for k := 0; k < uniqueKeys; k++ {
			key := "key-" + strconv.Itoa(k)
			for j := 0; j < groupSize; j++ {
				go func(k string) {
					<-start
					_, _, _ = g.Do(k, fn)
					done <- struct{}{}
				}(key)
			}
		}

		close(start) // все горутины стартуют одновременно

		for j := 0; j < total; j++ {
			<-done
		}
	}
}

func BenchmarkDo_1k_Grouped(b *testing.B) {
	runParallelDoGrouped(b, 1000, 10) // 100 уникальных ключей, каждая группа 10
}

func BenchmarkDo_10k_Grouped(b *testing.B) {
	runParallelDoGrouped(b, 10000, 10) // 1000 уникальных ключей
}

func BenchmarkDo_100k_Grouped(b *testing.B) {
	runParallelDoGrouped(b, 100000, 10) // 10000 уникальных ключей
}
