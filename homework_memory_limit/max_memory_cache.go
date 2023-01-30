package cache

import (
	"context"
	"sync"
	"time"

	"github.com/gotomicro/ekit/list"
)

type MaxMemoryCache struct {
	Cache
	max  int64
	used int64
	lock sync.Mutex
	list *list.LinkedList[string] // 队尾元素为 Get和Set时的 设置元素，  对首为最少使用元素，需要淘汰
}

func NewMaxMemoryCache(max int64, cache Cache) *MaxMemoryCache {
	res := &MaxMemoryCache{
		max:   max,
		used:  0,
		Cache: cache,
		list:  list.NewLinkedList[string](),
	}
	res.Cache.OnEvicted(func(key string, val []byte) {
		// 注册回调
		res.lock.Lock()
		defer res.lock.Unlock()

		res.used -= int64(len(val))

		for i := 0; i < res.list.Len(); i++ {
			item, _ := res.list.Get(i)
			if key == item {
				res.list.Delete(i)
			}
		}
	})

	return res
}

func (m *MaxMemoryCache) Set(ctx context.Context, key string, val []byte,
	expiration time.Duration) error {
	// 在这里判断内存使用量，以及腾出空间

	needSize := int64(len(val))

	m.lock.Lock()
	// 这里不能用defer， 否则 Delete在OnEvicted会拿不到锁

	used := m.used
	// 如果覆写，则要删除原来的size
	item, _ := m.Cache.Get(ctx, key)
	isExists := false
	if len(item) > 0 {
		used -= int64(len(item))
		isExists = true
	}

	if m.max < needSize+used {
		// 清理
		deleteKey := make([]string, 0, 2)
		cap := m.list.Len() - 1
		var free int64 = 0
		// 获得需要删除key
		for i := cap; i >= 0; i-- { // 从队尾开始删
			key, err := m.list.Get(i)
			if err != nil {
				m.lock.Unlock()
				return err
			}

			item, err := m.Cache.Get(ctx, key)
			if err != nil {
				m.lock.Unlock()
				return err
			}

			free += int64(len(item))
			deleteKey = append(deleteKey, key)

			if used-free > needSize {
				break
			}
		}

		m.lock.Unlock()

		for k := range deleteKey {
			m.Cache.Delete(ctx, deleteKey[k]) // 这里不需要去减used， 通过OnEvicated去做扣除即可
		}

	} else {
		// 不需要清理，直接加size
		m.used = m.used - int64(len(item)) + needSize // 先扣除原来的长度，再加新的，即使原来为0 也不影响结果
		m.lock.Unlock()
	}

	if !isExists { // 不存在就直接添加，减少遍历
		err := m.list.Add(0, key) // 队首为热数据
		if err != nil {
			return err
		}
	} else {
		err := listMoveToHead(m.list, key)
		if err != nil {
			return err
		}
	}

	return m.Cache.Set(ctx, key, val, expiration)
}

func (m *MaxMemoryCache) Get(ctx context.Context, key string) ([]byte, error) {

	v, err := m.Cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	//m.list.Add(0, key) // 对尾为热数据
	err = listMoveToHead(m.list, key)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func listMoveToHead(l *list.LinkedList[string], key string) error {
	// 这里假设key是存在的，直接要移动
	for i := 0; i < l.Len(); i++ {
		item, _ := l.Get(i)
		if key == item {
			_, err := l.Delete(i)
			if err != nil {
				return err
			}
			break
		}
	}
	return l.Add(0, key)
}
