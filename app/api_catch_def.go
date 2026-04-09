/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_catch_def
 *@date    2025/11/4 18:21
 */

package app

import (
	"container/list"
	"sync"
	"time"
)

type ExpiringCatchData struct {
	Data       catchdata
	Expiration time.Time
}

type ExpiringCatchDataSlice struct {
	data  *list.List
	mu    sync.RWMutex
	stop  chan struct{}
}

func NewExpiringCatchDataSlice() *ExpiringCatchDataSlice {
	ecds := &ExpiringCatchDataSlice{
		data: list.New(),
		stop: make(chan struct{}),
	}
	go ecds.backgroundCleanup()
	return ecds
}

func (ecds *ExpiringCatchDataSlice) Add(data catchdata) {
	ecds.mu.Lock()
	defer ecds.mu.Unlock()

	ecds.data.PushBack(ExpiringCatchData{
		Data:       data,
		Expiration: time.Now().Add(time.Minute),
	})
}

// Del 根据table名称删除所有匹配的元素
func (ecds *ExpiringCatchDataSlice) Del(table string) int {
	ecds.mu.Lock()
	defer ecds.mu.Unlock()

	count := 0
	var next *list.Element

	for e := ecds.data.Front(); e != nil; e = next {
		next = e.Next()
		item := e.Value.(ExpiringCatchData)
		if item.Data.Table == table {
			ecds.data.Remove(e)
			count++
		}
	}
	return count
}

func (ecds *ExpiringCatchDataSlice) GetAll() []catchdata {
	ecds.mu.RLock()
	defer ecds.mu.RUnlock()

	now := time.Now()
	var result []catchdata

	for e := ecds.data.Front(); e != nil; e = e.Next() {
		item := e.Value.(ExpiringCatchData)
		if item.Expiration.After(now) {
			result = append(result, item.Data)
		}
	}

	return result
}

// 按条件查询（例如按Table名称）
func (ecds *ExpiringCatchDataSlice) GetByTable(table string) []catchdata {
	ecds.mu.RLock()
	defer ecds.mu.RUnlock()

	now := time.Now()
	var result []catchdata

	for e := ecds.data.Front(); e != nil; e = e.Next() {
		item := e.Value.(ExpiringCatchData)
		if item.Expiration.After(now) && item.Data.Table == table {
			result = append(result, item.Data)
		}
	}

	return result
}

func (ecds *ExpiringCatchDataSlice) backgroundCleanup() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ecds.cleanupExpired()
		case <-ecds.stop:
			return
		}
	}
}

func (ecds *ExpiringCatchDataSlice) cleanupExpired() {
	ecds.mu.Lock()
	defer ecds.mu.Unlock()

	now := time.Now()
	var next *list.Element

	for e := ecds.data.Front(); e != nil; e = next {
		next = e.Next()
		item := e.Value.(ExpiringCatchData)
		if item.Expiration.Before(now) {
			ecds.data.Remove(e)
		}
	}
}

func (ecds *ExpiringCatchDataSlice) Stop() {
	close(ecds.stop)
}

// ResetExp 重置匹配table名称元素的过期时间（从当前时间重新计算）
// 返回重置的元素数量和已过期的数量
func (ecds *ExpiringCatchDataSlice) ResetExp(table string){
	ecds.mu.Lock()
	defer ecds.mu.Unlock()

	now := time.Now()
	newExpiration := now.Add(time.Minute)

	for e := ecds.data.Front(); e != nil; e = e.Next() {
		item := e.Value.(ExpiringCatchData)
		if item.Data.Table == table {
			if item.Expiration.After(now) {
				// 重置过期时间（从当前时间重新计算）
				e.Value = ExpiringCatchData{
					Data:       item.Data,
					Expiration: newExpiration,
				}
			}
		}
	}
}