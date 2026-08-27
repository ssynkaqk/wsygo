package Wsy

import (
	"net/http"
	"net/url"
	"sync"
	"time"
	"github.com/gorilla/websocket"
)

type WsyTLS struct {
	Port        string
	Path        string
	connections map[string]*WsyTLSConn
	mu          sync.RWMutex
	PongWait   int
	PingPeriod int
	WriteWait  int
	MaxMsgSize int64
}

type WsyTLSConn struct {
	Conn      *websocket.Conn
	ID        string
	Path      string
	Query     url.Values
	Send      chan []byte
	Recv      chan []byte
	Done      chan struct{}
	tls       *WsyTLS
	closeOnce sync.Once
	OnSwitch  func(path string, query url.Values)
}
//加载初始化
func (t *WsyTLS) LoadInit() {
	if t.connections == nil {
		t.connections = make(map[string]*WsyTLSConn)
	}
	t.PongWait   = Str.IIFS(t.PongWait != 0, t.PongWait, 10).(int)
	t.PingPeriod = Str.IIFS(t.PingPeriod != 0, t.PingPeriod, 5).(int)
	t.WriteWait  = Str.IIFS(t.WriteWait != 0, t.WriteWait, 3).(int)
	t.MaxMsgSize = Str.IIFS(t.MaxMsgSize != 0, t.MaxMsgSize, int64(4096)).(int64)
}
//启动服务
func (t *WsyTLS) New(fn func(conn *WsyTLSConn, boxid string)) {
	t.LoadInit()
	t.Port = Str.IIF(t.Port == "", "56810", t.Port)
	t.Path = Str.IIF(t.Path == "", "/", t.Path)
	http.HandleFunc(t.Path, func(w http.ResponseWriter, r *http.Request) {
		boxid := r.URL.Query().Get("boxid")
		if boxid == "" {
			boxid = "unknown"
		}
		conn, err := t.Upgrade(w, r, boxid)
		if err != nil {
			Logs("ERROR", "WSYTLS", "升级WebSocket失败: "+err.Error(), "Y")
			return
		}
		conn.Path = r.URL.Path
		conn.Query = r.URL.Query()
		fn(conn, boxid)
	})
	Logs("INFO", "WSYTLS", "WebSocket服务启动, 端口:"+ t.Port +" 路由:" + t.Path, "Y")
	if err := http.ListenAndServe(":"+t.Port, nil); err != nil {
		Logs("ERROR", "WSYTLS", "WebSocket服务启动失败: "+err.Error(), "Y")
	}
}
//升级WebSocket
func (t *WsyTLS) Upgrade(w http.ResponseWriter, r *http.Request, id string) (*WsyTLSConn, error) {
	upgrader := t.NewUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	c := &WsyTLSConn{
		Conn: conn,
		ID:   id,
		Send: make(chan []byte, 64),
		Recv: make(chan []byte, 64),
		Done: make(chan struct{}),
		tls:  t,
	}
	t.mu.Lock()
	t.connections[id] = c
	t.mu.Unlock()
	go c.writeLoop()
	go c.readLoop()
	return c, nil
}
//升级WebSocket
func (t *WsyTLS) NewUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
//获取连接
func (t *WsyTLS) GetConn(id string) *WsyTLSConn {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connections[id]
}
//删除连接
func (t *WsyTLS) DelConn(id string, c *WsyTLSConn) {
	t.mu.Lock()
	if t.connections[id] == c {
		delete(t.connections, id)
	}
	t.mu.Unlock()
}
//广播信息
func (t *WsyTLS) Broadcast(msg []byte) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, c := range t.connections {
		select {
		case c.Send <- msg:
		default:
		}
	}
}
//获取连接数量
func (t *WsyTLS) ConnCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.connections)
}
//发送信息
func (c *WsyTLSConn) SendText(msg string) {
	c.Send <- []byte(msg)
}
//运行时切换 path 和 query（服务端 conn.Path/Query 同步更新）
func (c *WsyTLSConn) Switch(path string, query string) {
	c.Send <- []byte("__SW__" + path + "|" + query)
}
//接收信息
func (c *WsyTLSConn) RecvLoop(fn func(msg []byte)) {
	for msg := range c.Recv {
		fn(msg)
	}
}
//定时发送信息
func (c *WsyTLSConn) SendEvery(sec int, fn func() string) {
	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.SendText(fn())
		case <-c.Done:
			return
		}
	}
}
//会话循环：onRecv接收消息回调
func (c *WsyTLSConn) Session(onRecv func([]byte)) {
	for {
		select {
		case msg, ok := <-c.Recv:
			if !ok {
				return
			}
			msgStr := string(msg)
			if len(msgStr) > 6 && msgStr[:6] == "__SW__" {
				payload := msgStr[6:]
				for i := 0; i < len(payload); i++ {
					if payload[i] == '|' {
						c.Path = payload[:i]
						c.Query, _ = url.ParseQuery(payload[i+1:])
						break
					}
				}
				if c.OnSwitch != nil {
					c.OnSwitch(c.Path, c.Query)
				}
				continue
			}
			onRecv(msg)
		case <-c.Done:
			return
		}
	}
}
//关闭连接
func (c *WsyTLSConn) Close() {
	c.closeOnce.Do(func() {
		c.tls.DelConn(c.ID, c)
		close(c.Done)
		c.Conn.Close()
	})
}
//读取信息
func (c *WsyTLSConn) readLoop() {
	defer func() { c.Close() }()
	c.Conn.SetReadLimit(c.tls.MaxMsgSize)
	c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.tls.PongWait) * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(time.Duration(c.tls.PongWait) * time.Second))
		return nil
	})
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		select {
		case c.Recv <- msg:
		case <-c.Done:
			return
		}
	}
}
//写入信息
func (c *WsyTLSConn) writeLoop() {
	ticker := time.NewTicker(time.Duration(c.tls.PingPeriod) * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.tls.WriteWait) * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(time.Duration(c.tls.WriteWait) * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.Done:
			return
		}
	}
}

//客户端连接（内置断线重连）
func (t *WsyTLS) Client(url string, id string, fn func(conn *WsyTLSConn)) {
	t.LoadInit()
	url = Str.IIF(url == "", "ws://127.0.0.1:56810/ws", url)
	id = Str.IIF(id == "", "client", id)
	for {
		wsConn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			Logs("ERROR", "WSYTLS", "WebSocket客户端连接失败: "+err.Error()+", 1秒后重试", "Y")
			time.Sleep(1 * time.Second)
			continue
		}
		c := &WsyTLSConn{
			Conn: wsConn,
			ID:   id,
			Send: make(chan []byte, 64),
			Recv: make(chan []byte, 64),
			Done: make(chan struct{}),
			tls:  t,
		}
		t.mu.Lock()
		t.connections[id] = c
		t.mu.Unlock()
		Logs("INFO", "WSYTLS", "WebSocket客户端已连接: "+url, "Y")
		go c.writeLoop()
		go c.readLoop()
		fn(c)
		Logs("INFO", "WSYTLS", "连接断开，1秒后重连...", "Y")
		time.Sleep(1 * time.Second)
	}
}