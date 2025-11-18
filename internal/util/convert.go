package util

import (
	"hash/fnv"
	"math/rand"
	"slices"
	"strconv"
	"time"
)

func BytesToInt64(b []byte) (int64, error) {
	result, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func Int64ToBytes(n int64) []byte {
	return []byte(strconv.FormatInt(n, 10))
}

func SetExpiration(defaultTTL, maxTTL int64, reqTTL int64) (expiration time.Duration, persistent bool) {
	var ttl int64
	persistent = false
	if reqTTL < 0 {
		ttl = -1
		persistent = true
	} else if reqTTL == 0 {
		ttl = defaultTTL
	} else if reqTTL > maxTTL {
		ttl = maxTTL
	} else {
		ttl = reqTTL
	}
	expiration = time.Duration(ttl) * time.Second
	return expiration, persistent
}

// key to hash
func Fnv32aHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// 최대 샤드 개수 중 n개의 샤드 인덱스 반환
func GetRandomShardIndex(shardCount int, count int) []int {
	if count <= 0 || count > shardCount {
		return nil
	}
	p := rand.Perm(shardCount)
	indexes := p[:count]
	// sort indexes to have consistent order
	slices.Sort(indexes)
	return indexes
}

func GetIndexListNoDup(keys []string, getIndexFunc func(string) int) []int {
	indexList := make([]int, 0, len(keys))
	indexSet := make(map[int]struct{})
	for _, key := range keys {
		index := getIndexFunc(key)
		if _, exists := indexSet[index]; !exists {
			indexSet[index] = struct{}{}
			indexList = append(indexList, index)
		}
	}
	slices.Sort(indexList)
	return indexList
}
