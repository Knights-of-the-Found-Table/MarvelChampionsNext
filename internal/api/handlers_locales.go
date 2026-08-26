package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/Knights-of-the-Found-Table/marvelchampionsnext/internal/engine"
)

// i18n 文案分发（唯一事实源是仓库根 i18n/messages.json，经根级 i18n 包
// //go:embed 嵌入服务端）。三条路由，均为公开端点（目录不含用户数据）：
//
//	  GET /api/v1/locales/manifest       -> {"en":"<hash16>","zh":"<hash16>"}
//
//		no-cache：客户端每次启动回源校验；载荷极小，hash 变了才需要
//		重新拉目录。
//
//	  GET /api/v1/locales/{lang}/{hash}  -> 目录正文（内容寻址）
//
//		hash 与当前内容一致才 200 并打 immutable 长缓存头——URL 内容
//		不变即正文不变，浏览器可长期复用磁盘缓存；hash 失配（客户端
//		缓存的目录比服务端二进制旧/新）返回 404，客户端重取 manifest
//		换新 URL 自愈。
//
//	  GET /api/v1/locales/{lang}         -> 兼容旧客户端的直连路由
//
//		max-age=300 短缓存，响应正文与前两者相同。
const (
	localeHashLen   = 16
	localeMaxAge    = "max-age=300"                         // 兼容直连路由：短缓存
	localeImmutable = "public, max-age=31536000, immutable" // 内容寻址：长缓存
	manifestPolicy  = "no-cache"                            // manifest 永远校验
)

var (
	localeOnce sync.Once
	localeBody map[engine.Lang][]byte
	localeHash map[engine.Lang]string
)

// localeContent 把目录按语言序列化并计算内容 hash，进程内只算一次。
// json.Marshal 对 map 按键排序，输出确定——hash 即内容地址。目录随
// 二进制静态嵌入，运行期永不变化。
func localeContent() {
	localeOnce.Do(func() {
		localeBody = make(map[engine.Lang][]byte, 2)
		localeHash = make(map[engine.Lang]string, 2)
		for _, l := range []engine.Lang{engine.LangEn, engine.LangZh} {
			b, err := json.Marshal(engine.Messages(l))
			if err != nil { // map[string]string 不会失败；防未来改动
				panic("i18n: marshal locale: " + err.Error())
			}
			sum := sha256.Sum256(b)
			localeBody[l] = b
			localeHash[l] = hex.EncodeToString(sum[:])[:localeHashLen]
		}
	})
}

func writeLocale(w http.ResponseWriter, status int, l engine.Lang) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(localeBody[l])
}

// handleLocalesManifest 供客户端启动时获取各语言目录的内容地址。
func (s *Server) handleLocalesManifest(w http.ResponseWriter, r *http.Request) {
	localeContent()
	w.Header().Set("Cache-Control", manifestPolicy)
	writeJSON(w, http.StatusOK, map[string]string{
		string(engine.LangEn): localeHash[engine.LangEn],
		string(engine.LangZh): localeHash[engine.LangZh],
	})
}

// handleLocalesVersioned 服务内容寻址的目录正文；未知语言或失配的 hash
// 都是 404（客户端据此重取 manifest 自愈）。
func (s *Server) handleLocalesVersioned(w http.ResponseWriter, r *http.Request) {
	lang := engine.Lang(r.PathValue("lang"))
	localeContent()
	body, ok := localeBody[lang]
	if !ok || r.PathValue("hash") != localeHash[lang] {
		writeErr(w, http.StatusNotFound, "unknown locale or stale hash")
		return
	}
	w.Header().Set("Cache-Control", localeImmutable)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// handleLocales 是迁移前就存在的直连路由：/locales/{lang} 直接给当前
// 目录，短缓存；为不带 hash 的旧客户端保留。
func (s *Server) handleLocales(w http.ResponseWriter, r *http.Request) {
	lang := engine.Lang(r.PathValue("lang"))
	localeContent()
	if _, ok := localeBody[lang]; !ok {
		writeErr(w, http.StatusNotFound, "unknown locale")
		return
	}
	w.Header().Set("Cache-Control", localeMaxAge)
	writeLocale(w, http.StatusOK, lang)
}
