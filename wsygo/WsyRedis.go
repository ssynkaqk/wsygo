package Wsy

import (
	"sync"
	"time"
	"errors"
	"context"
	"strings"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

/*
// 方式一：手动配置
Wsy.Redis.Config(Wsy.RedisConfig{
    Host:     "127.0.0.1",
    Port:     "6379",
    PassWord: "Aa@linqijing2026",
    DB:       "0,1,2,3",          // 逗号分隔，同时连接多个库
})
Wsy.Redis.Init()

// 方式二：从 ini 加载
Wsy.Redis.LoadInit("/path/webapi.ini", "Redis")
Wsy.Redis.Init()
// 分组读写删
Wsy.Redis.DB(2).Sets("DEVOS_TIME_UID", uid, value, 900)    // DB 2 写入
value, ok := Wsy.Redis.DB(2).Gets("DEVOS_TIME_UID", uid)    // DB 2 读取
Wsy.Redis.DB(2).Dels("DEVOS_TIME_UID", uid)                  // DB 2 删除

// 普通键操作
Wsy.Redis.DB(0).Set("key", "hello", 3600)
val, ok := Wsy.Redis.DB(0).Get("key")
Wsy.Redis.DB(0).Del("key")

// 获取原生客户端
client := Wsy.Redis.DB(3).GetClient()
*/

// WsyRedis 提供了轻量级的 Redis 操作封装（全局单例，负责连接池；并发读写请用 DB(n) 返回的句柄）
type WsyRedis struct {
	clients     map[int]*redis.Client
	defaultDB   int
	config      RedisConfig
	dbs         []int
	once        sync.Once
	mu          sync.RWMutex
	Debug       bool
	DialTimeout int // 建立 TCP 连接超时（秒），默认 2
	OpTimeout   int // 读写命令超时（秒），默认 3
	InitTimeout int // Init 时 Ping 超时（秒），默认 3
	PoolTimeout int // 从连接池取连接的最长等待（秒），默认 5
}

// WsyRedisDB 指定库编号的操作句柄，可安全并发（不修改全局状态）
type WsyRedisDB struct {
	r  *WsyRedis
	db int
}
// RedisConfig Redis配置结构体
type RedisConfig struct {
	DB       string // 数据库编号，支持 "0"、"DB0"、多库 "0,1,2,3" 或 "DB0,DB1,DB2"
	Host     string // 主机，如 127.0.0.1；也可写 127.0.0.1:6379（此时忽略 Port）
	Port     string // 端口，默认 6379；Config 时自动拼到 Host
	PassWord string // Redis密码
}
// SetInit 初始化超时等默认值（秒）
func (r *WsyRedis) SetInit() {
	r.DialTimeout = Str.IIFS(r.DialTimeout != 0, r.DialTimeout, int(2)).(int)
	r.OpTimeout   = Str.IIFS(r.OpTimeout   != 0, r.OpTimeout, int(3)).(int)
	r.InitTimeout = Str.IIFS(r.InitTimeout != 0, r.InitTimeout, int(3)).(int)
	r.PoolTimeout = Str.IIFS(r.PoolTimeout != 0, r.PoolTimeout, int(5)).(int)
}

// Config 设置全局配置（Host+Port 拼接，DB 支持 "0" / "DB0" / "0,1,2,3" / "DB0,DB1"）
func (r *WsyRedis) Config(cfg RedisConfig) {
	host := strings.TrimSpace(cfg.Host)
	if host != "" && !strings.Contains(host, ":") {
		p := strings.TrimSpace(cfg.Port)
		if p == "" {
			p = "6379"
		}
		host += ":" + p
	}
	cfg.Host, cfg.Port = host, ""

	dbStr := strings.TrimSpace(cfg.DB)
	dbParts := strings.Split(dbStr, ",")
	dbs := make([]int, 0, len(dbParts))
	for _, part := range dbParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(part), "DB") {
			part = strings.TrimSpace(part[2:])
		}
		dbs = append(dbs, Str.ToInt(part, 0))
	}
	if len(dbs) == 0 {
		dbs = []int{0}
	}
	r.dbs = dbs
	r.defaultDB = dbs[0]
	r.config = cfg
}
// LoadInit 从 ini 加载 Redis 配置（支持 KeyData 加密）
// 示例：Wsy.Redis.LoadInit("/path/webapi.ini", "Redis")
func (r *WsyRedis) LoadInit(iniPath ...string) (RedisConfig, error) {
	var cfg RedisConfig
	var mconf map[string]string
	var jsonStr string
	configPath := Set.File
	if len(iniPath) > 0 && iniPath[0] != "" {
		configPath = iniPath[0]
	}
	sectionName := "Redis"
	if len(iniPath) > 1 && iniPath[1] != "" {
		sectionName = iniPath[1]
	}
	enc := Fso.ReadIni(configPath, sectionName, "KeyData")
	if enc != "" {
		jsonStr = Key.DeCode(enc)
		if jsonStr == "" {
			return cfg, errors.New("Redis配置解密失败")
		}
	} else {
		jsonStr = Fso.ReadIni(configPath, sectionName)
		if jsonStr == "" {
			return cfg, errors.New("未找到Redis配置")
		}
	}
	if err := json.Unmarshal([]byte(jsonStr), &mconf); err != nil {
		return cfg, errors.New("Redis配置反序列化失败: " + err.Error())
	}
	cfg = RedisConfig{
		Host:     mconf["Host"],
		Port:     mconf["Port"],
		PassWord: mconf["PassWord"],
		DB:       mconf["DB"],
	}
	if cfg.Host == "" {
		return cfg, errors.New("未找到Redis Host配置")
	}
	r.Config(cfg)
	return r.config, nil
}

// DB 返回指定库的操作句柄（并发安全，每次调用独立句柄）
func (r *WsyRedis) DB(db int) *WsyRedisDB {
	return &WsyRedisDB{r: r, db: db}
}

// Open 检查默认库连接是否可用，不可用则自动 Init
func (r *WsyRedis) Open() error {
	return r.DB(r.defaultDB).Open()
}

// Close 关闭所有 DB 的 Redis 连接
func (r *WsyRedis) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.clients == nil {
		return nil
	}
	var lastErr error
	for db, client := range r.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
		delete(r.clients, db)
	}
	return lastErr
}

// Init 初始化Redis连接，为每个配置的 DB 创建独立客户端
func (r *WsyRedis) Init() error {
	var redisConf RedisConfig
	if r.config != (RedisConfig{}) {
		redisConf = r.config
	} else {
		Logs("INFO", "Redis", "Redis全局配置未设置，请先调用Config方法")
		return errors.New("Redis全局配置未设置，请先调用Config方法")
	}
	r.SetInit()
	var initErr error
	r.once.Do(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.clients = make(map[int]*redis.Client, len(r.dbs))
		for _, db := range r.dbs {
			client := redis.NewClient(&redis.Options{
				Addr:         redisConf.Host,
				Password:     redisConf.PassWord,
				DB:           db,
				DialTimeout:  time.Duration(r.DialTimeout) * time.Second,
				ReadTimeout:  time.Duration(r.OpTimeout) * time.Second,
				WriteTimeout: time.Duration(r.OpTimeout) * time.Second,
				PoolTimeout:  time.Duration(r.PoolTimeout) * time.Second,
				MaxRetries:   0,
				MaintNotificationsConfig: &maintnotifications.Config{
					Mode: maintnotifications.ModeDisabled,
				},
				DisableIndentity: true,
			})
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.InitTimeout)*time.Second)
			if err := client.Ping(ctx).Err(); err != nil {
				cancel()
				client.Close()
				Logs("Error", "Redis", "Redis DB"+Str.ToString(db)+"连接失败: "+err.Error())
				initErr = errors.New("Redis DB" + Str.ToString(db) + "连接失败: " + err.Error())
				return
			}
			cancel()
			r.clients[db] = client
		}
	})
	r.mu.RLock()
	clients := r.clients
	r.mu.RUnlock()
	if clients == nil || len(clients) == 0 {
		return initErr
	}
	return nil
}



// IsInit 检查 Redis 是否已初始化
func (r *WsyRedis) IsInit() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients) > 0
}

// Open 检查当前句柄对应库的连接
func (h *WsyRedisDB) Open() error {
	client := h.getClient()
	if client == nil {
		return h.r.Init()
	}
	h.r.SetInit()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.r.OpTimeout)*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return h.r.Init()
	}
	return nil
}

func (h *WsyRedisDB) getClient() *redis.Client {
	h.r.mu.RLock()
	defer h.r.mu.RUnlock()
	if h.r.clients == nil {
		return nil
	}
	return h.r.clients[h.db]
}

func (h *WsyRedisDB) ctx() (context.Context, context.CancelFunc) {
	h.r.SetInit()
	return context.WithTimeout(context.Background(), time.Duration(h.r.OpTimeout)*time.Second)
}

// GetClient 获取当前库的 go-redis 客户端（客户端本身支持并发）
func (h *WsyRedisDB) GetClient() *redis.Client {
	return h.getClient()
}

// Set 设置键值对，带过期时间（秒）
func (h *WsyRedisDB) Set(key string, value interface{}, ttl int) error {
	client := h.getClient()
	if client == nil {
		return errors.New("Redis未初始化，请先调用Open方法")
	}
	ctx, cancel := h.ctx()
	defer cancel()
	if ttl > 0 {
		return client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
	}
	return client.Set(ctx, key, value, 0).Err()
}

// Get 获取键值
func (h *WsyRedisDB) Get(key string) (string, bool) {
	client := h.getClient()
	if client == nil {
		return "", false
	}
	ctx, cancel := h.ctx()
	defer cancel()
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return val, true
}

// Del 删除键
func (h *WsyRedisDB) Del(key string) error {
	client := h.getClient()
	if client == nil {
		return errors.New("Redis未初始化，请先调用Open方法")
	}
	ctx, cancel := h.ctx()
	defer cancel()
	return client.Del(ctx, key).Err()
}

// Key 生成分组的键名
func (h *WsyRedisDB) Key(group, key string) string {
	return group + ":" + key
}

// Sets 分组写入
func (h *WsyRedisDB) Sets(group, key string, value interface{}, ttl int) error {
	return h.Set(h.Key(group, key), value, ttl)
}

// Gets 分组读取
func (h *WsyRedisDB) Gets(group, key string) (string, bool) {
	return h.Get(h.Key(group, key))
}

// Dels 删除分组键
func (h *WsyRedisDB) Dels(group, key string) error {
	return h.Del(h.Key(group, key))
}