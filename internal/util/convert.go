package util

import (
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
