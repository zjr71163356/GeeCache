package geecache

import (
	"GeeCache/consistenthash"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool 作为一个 HTTP 服务端，负责处理节点间的通信。
type HTTPPool struct {
	self        string                 // 记录自己的地址，包括主机名/IP和端口
	basePath    string                 // 作为节点间通讯地址的前缀，默认为 /_geecache/
	mu          sync.Mutex             //锁机制，并发安全
	peers       *consistenthash.Map    //一致性哈希结构体
	httpGetters map[string]*httpGetter //通过节点的名称作为键找到httpGetter的地址
}

// httpGetter 属于PeerGetter接口的类型，Pickpeer通过key获取节点返回PeerGetter，即可以返回httpGetter
type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {

	newUrl := fmt.Sprintf("%v%v/%v", h.baseURL,
		url.QueryEscape(group), url.QueryEscape(key),
	)

	rsp, err := http.Get(newUrl)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:%v", rsp.StatusCode)
	}

	bytes, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body:%v", err)
	}

	return bytes, nil
}

// NewHTTPPool 创建一个新的 HTTPPool 实例。
//
// 此函数用于初始化一个 HTTPPool，它将作为分布式缓存节点间的通信服务端。
//
// 参数:
//
//	self: 当前节点的地址，例如 "localhost:8001"。
//	basePath: 节点间通信的基础路径前缀。
//
// 返回值:
//
//	*HTTPPool: 一个指向新创建的 HTTPPool 实例的指针。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Set updates the pool's list of peers.
func (h *HTTPPool) Set(peers ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers = consistenthash.New(defaultReplicas, nil)
	h.peers.Add(peers...)

	h.httpGetters = make(map[string]*httpGetter)
	for _, peer := range peers {
		h.httpGetters[peer] = &httpGetter{
			baseURL: peer + h.basePath,
		}
	}

}

// PickPeer picks a peer according to key
func (h *HTTPPool) PickPeer(key string) (PeerGetter, bool) {

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.peers == nil {
		h.Log("HTTPPool peers is nil")
		return nil, false
	}

	if peer := h.peers.Get(key); peer != "" && peer != h.self {
		h.Log("Pick peer %s", peer)
		return h.httpGetters[peer], true
	}

	return nil, false

}

// Log 是一个日志记录辅助方法。
//
// 它会在日志消息前加上服务器的地址（self 字段），
// 方便在查看多个节点的聚合日志时区分日志来源。
//
// 参数:
//
//	format: 日志消息的格式化字符串。
//	a:      格式化字符串对应的可变参数。
func (h *HTTPPool) Log(format string, a ...any) {
	log.Printf("[Server %s]%s", h.self, fmt.Sprintf(format, a...))
}

// ServeHTTP 实现了 http.Handler 接口，用于处理 HTTP 请求。
//
// 它的核心功能是解析请求路径，格式应为 /<basepath>/<groupname>/<key>。
// 它会验证路径前缀，然后提取 group 名称和 key。
// 之后，它会从对应的 group 中获取缓存数据，并将其作为 HTTP 响应返回。
// 如果发生任何错误（如路径格式错误、group 不存在），它会返回相应的 HTTP 错误码。
//
// 参数:
//
//	w: 用于写入 HTTP 响应的 http.ResponseWriter。
//	r: 代表客户端发来的 HTTP 请求的 *http.Request。
func (h *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !strings.HasPrefix(r.URL.Path, h.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	h.Log("%s %s", r.Method, r.URL.Path)
	// 期望的请求路径格式为 /<basepath>/<groupname>/<key>
	// 使用 SplitN 将路径切分为两部分
	parts := strings.SplitN(r.URL.Path[len(h.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 将获取到的缓存值作为二进制流写入响应体
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
