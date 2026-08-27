package Wsy

import (
	"sync"
	"time"
)
// CacheItem 缓存项结构
type CacheItem struct {
	Value      interface{} // 缓存值
	ExpireTime time.Time   // 过期时间
}
// WsyCache 缓存管理结构
type WsyCache struct {
	items     map[string]*CacheItem // 缓存存储
	mutex     sync.RWMutex          // 读写锁
	once      sync.Once              // 用于延迟初始化
	cleanupOnce sync.Once           // 用于启动清理协程
}
// initCache 延迟初始化缓存
func (c *WsyCache) initCache() {
	c.once.Do(func() {
		if c.items == nil {
			c.items = make(map[string]*CacheItem)
		}
		// 启动后台清理协程（只启动一次）
		c.cleanupOnce.Do(func() {
			go c.cleanup()
		})
	})
}

// Set 设置缓存，带过期时间（秒）
// key: 缓存键
// value: 缓存值
// ttl: 过期时间（秒），0表示不过期
func (c *WsyCache) Set(key string, value interface{}, ttl int) {
	c.initCache()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	item := &CacheItem{
		Value: value,
	}
	// 设置过期时间
	if ttl > 0 {
		item.ExpireTime = time.Now().Add(time.Duration(ttl) * time.Second)
	} else {
		// ttl为0或负数，设置一个很远的未来时间，表示永不过期
		item.ExpireTime = time.Now().Add(365 * 24 * time.Hour)
	}
	c.items[key] = item
}
// Get 获取缓存值
// key: 缓存键
// 返回：缓存值和是否存在
func (c *WsyCache) Get(key string) (interface{}, bool) {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	// 检查是否过期
	if time.Now().After(item.ExpireTime) {
		// 已过期，异步删除
		go func() {
			c.mutex.Lock()
			delete(c.items, key)
			c.mutex.Unlock()
		}()
		return nil, false
	}
	return item.Value, true
}
// Del 删除缓存
// key: 缓存键
func (c *WsyCache) Del(key string) {
	c.initCache()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.items, key)
}
// Exists 检查缓存是否存在且未过期
// key: 缓存键
// 返回：是否存在且未过期
func (c *WsyCache) Exists(key string) bool {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	item, exists := c.items[key]
	if !exists {
		return false
	}
	// 检查是否过期
	if time.Now().After(item.ExpireTime) {
		return false
	}
	return true
}

// Expire 检查缓存是否未过期
// key: 缓存键
// 返回：false表示已过期或不存在，true表示未过期
func (c *WsyCache) Expire(key string) bool {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	item, exists := c.items[key]
	if !exists {
		return false // 不存在
	}
	// 检查是否过期
	if time.Now().After(item.ExpireTime) {
		return false // 已过期
	}
	return true // 未过期
}

// cleanup 后台清理过期缓存的协程
func (c *WsyCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		if c.items == nil {
			c.mutex.Unlock()
			continue
		}
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpireTime) {
				delete(c.items, key)
			}
		}
		c.mutex.Unlock()
	}
}

// Clear 清空所有缓存
func (c *WsyCache) Clear() {
	c.initCache()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]*CacheItem)
}

// Count 获取缓存数量
func (c *WsyCache) Count() int {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}
////////////////////////////////////////////////////////////////////////////////////////分组方法//////////////////////
// buildGroupKey 构建分组键名
func (c *WsyCache) buildGroupKey(group string, key string) string {
	return group + ":" + key
}

// Sets 设置分组缓存
// group: 组名（如 "uidlist"）
// key: 组内键
// value: 缓存值
// ttl: 过期时间（秒），0表示不过期
// 示例：Wsy.Cache.Sets("uidlist", "1", "json数据", 30)
func (c *WsyCache) Sets(group string, key string, value interface{}, ttl int) {
	cacheKey := c.buildGroupKey(group, key)
	c.Set(cacheKey, value, ttl)
}

// Gets 获取分组缓存值
// group: 组名
// key: 组内键
// 返回：缓存值和是否存在
// 示例：value, exists := Wsy.Cache.Gets("uidlist", "1")
func (c *WsyCache) Gets(group string, key string) (interface{}, bool) {
	cacheKey := c.buildGroupKey(group, key)
	return c.Get(cacheKey)
}

// Dels 删除分组中的指定缓存
// group: 组名
// key: 组内键
// 示例：Wsy.Cache.Dels("uidlist", "1")
func (c *WsyCache) Dels(group string, key string) {
	cacheKey := c.buildGroupKey(group, key)
	c.Del(cacheKey)
}

// DelsAll 删除整个组的所有缓存
// group: 组名
// 示例：Wsy.Cache.DelsAll("uidlist") - 删除所有 uidlist:* 的缓存
func (c *WsyCache) DelsAll(group string) {
	c.initCache()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	prefix := group + ":"
	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

// GetKeys 获取组内所有键名
// group: 组名
// 返回：组内所有键的字符串列表
// 示例：keys := Wsy.Cache.GetKeys("uidlist") - 返回 ["1", "2", "3"]
func (c *WsyCache) GetsKey(group string) []string {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	prefix := group + ":"
	keys := make([]string, 0)
	for key, item := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// 检查是否过期
			if time.Now().Before(item.ExpireTime) {
				// 提取组内键（去掉组名前缀）
				groupKey := key[len(prefix):]
				keys = append(keys, groupKey)
			}
		}
	}
	return keys
}

// Counts 获取组内缓存数量
// group: 组名
// 返回：组内缓存项数量
func (c *WsyCache) Counts(group string) int {
	c.initCache()
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	prefix := group + ":"
	count := 0
	now := time.Now()
	for key, item := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			if now.Before(item.ExpireTime) {
				count++
			}
		}
	}
	return count
}

// Check 检查分组缓存是否存在且未过期
// group: 组名
// key: 组内键
// 返回：是否存在且未过期
// 示例：Wsy.Cache.Check("uidlist", "1")
func (c *WsyCache) Check(group string, key string) bool {
	cacheKey := c.buildGroupKey(group, key)
	return c.Exists(cacheKey)
}

// Expires 检查分组缓存是否未过期
// group: 组名
// key: 组内键
// 返回：false表示已过期或不存在，true表示未过期
// 示例：isValid := Wsy.Cache.Expires("uidlist", "1")
func (c *WsyCache) Expires(group string, key string) bool {
	cacheKey := c.buildGroupKey(group, key)
	return c.Expire(cacheKey)
}

//读取ini格式的配置
//Wsy.Cache.Sets("WEBNAV", "CONFIG", Config, 0)
//Wsy.Cache.GetIni("WEBNAV", "CONFIG", "Server", "FileServer") // 使用Gets(group,key)
//Wsy.Cache.GetIni("CONFIG", "Server", "FileServer")          // 使用Get(key)
func (c *WsyCache) GetIni(a, b, c2 string, d ...string) string {
	getIniFromCached := func(cachedData interface{}, section, valueKey string) string {
		sections, ok := cachedData.(map[string]map[string]string)
		if !ok { return "" }
		sectionLower := Str.ToLower(section)
		keyLower := Str.ToLower(valueKey)
		for sectionName, sectionConfig := range sections {
			if Str.ToLower(sectionName) == sectionLower {
				for keyName, val := range sectionConfig {
					if Str.ToLower(keyName) == keyLower {
						return val
					}
				}
			}
		}
		return ""
	}
	switch len(d) {
		case 0:
			if cachedData, ok := c.Get(a); ok { return getIniFromCached(cachedData, b, c2) }
		case 1:
			if cachedData, ok := c.Gets(a, b); ok { return getIniFromCached(cachedData, c2, d[0]) }
	}
	return ""
}
