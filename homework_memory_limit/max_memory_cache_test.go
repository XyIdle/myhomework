package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gotomicro/ekit/list"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxMemoryCache_Set(t *testing.T) {

	testCases := []struct {
		name           string
		before         func(t *testing.T, c Cache)
		after          func(t *testing.T, c Cache)
		expiration     time.Duration
		key            string
		val            []byte
		max            int64
		wantLinkedList *list.LinkedList[string]
		wantErr        error
	}{
		{
			name: "set success and no need evicted",
			before: func(t *testing.T, c Cache) {
				ctx := context.Background()
				for i := 0; i <= 9; i++ {
					err := c.Set(ctx, fmt.Sprintf("%d", i), []byte("0123456789"), time.Minute)
					require.NoError(t, err)
				}
			},
			after:          func(t *testing.T, c Cache) {},
			expiration:     time.Minute,
			key:            "key1",
			val:            []byte("01234"),
			max:            110,
			wantLinkedList: list.NewLinkedListOf[string]([]string{"key1", "9", "8", "7", "6", "5", "4", "3", "2", "1", "0"}),
		},
		{
			name: "set success and need evicted",
			before: func(t *testing.T, c Cache) {
				ctx := context.Background()
				for i := 0; i <= 9; i++ {
					err := c.Set(ctx, fmt.Sprintf("%d", i), []byte("0123456789"), time.Minute)
					require.NoError(t, err)
				}
			},
			after:          func(t *testing.T, c Cache) {},
			expiration:     time.Minute,
			key:            "key1",
			val:            []byte("01234"),
			max:            103,
			wantLinkedList: list.NewLinkedListOf[string]([]string{"key1", "9", "8", "7", "6", "5", "4", "3", "2", "1"}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cache := NewMemoryMapCache(time.Minute)
			maxMemCache := NewMaxMemoryCache(tc.max, cache)

			tc.before(t, maxMemCache)

			// for i := 0; i <= 9; i++ {
			// 	fmt.Println(maxMemCache.list.Get(i))
			// }

			ctx := context.Background()
			err := maxMemCache.Set(ctx, tc.key, tc.val, time.Minute)

			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantLinkedList.Len(), maxMemCache.list.Len())

			assert.Equal(t, tc.wantLinkedList.AsSlice(), maxMemCache.list.AsSlice())

			tc.after(t, cache)
		})
	}
}

func TestMaxMemoryCache_Get(t *testing.T) {

	testCases := []struct {
		name           string
		before         func(t *testing.T, c Cache)
		after          func(t *testing.T, c Cache)
		expiration     time.Duration
		key            string
		val            []byte
		max            int64
		wantLinkedList *list.LinkedList[string]
		wantErr        error
	}{
		{
			name: "get and move to head",
			before: func(t *testing.T, c Cache) {
				ctx := context.Background()
				for i := 0; i <= 9; i++ {
					err := c.Set(ctx, fmt.Sprintf("%d", i), []byte("0123456789"), time.Minute)
					require.NoError(t, err)
				}
			},
			after:          func(t *testing.T, c Cache) {},
			expiration:     time.Minute,
			key:            "5",
			val:            []byte("0123456789"),
			max:            110,
			wantLinkedList: list.NewLinkedListOf[string]([]string{"5", "9", "8", "7", "6", "4", "3", "2", "1", "0"}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cache := NewMemoryMapCache(time.Minute)
			maxMemCache := NewMaxMemoryCache(tc.max, cache)

			tc.before(t, maxMemCache)

			// for i := 0; i <= 9; i++ {
			// 	fmt.Println(maxMemCache.list.Get(i))
			// }

			ctx := context.Background()
			val, err := maxMemCache.Get(ctx, tc.key)

			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.val, val)

			assert.Equal(t, tc.wantLinkedList.AsSlice(), maxMemCache.list.AsSlice())

			//fmt.Println(maxMemCache.list.AsSlice())

			tc.after(t, cache)
		})
	}
}
