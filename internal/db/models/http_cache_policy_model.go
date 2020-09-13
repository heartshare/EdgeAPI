package models

//
type HTTPCachePolicy struct {
	Id                       uint32 `field:"id"`                       // ID
	AdminId                  uint32 `field:"adminId"`                  // 管理员ID
	UserId                   uint32 `field:"userId"`                   // 用户ID
	IsOn                     uint8  `field:"isOn"`                     // 是否启用
	Name                     string `field:"name"`                     // 名称
	Key                      string `field:"key"`                      // 缓存Key规则
	Capacity                 string `field:"capacity"`                 // 容量数据
	Life                     string `field:"life"`                     // 有效期
	Status                   string `field:"status"`                   // HTTP状态码列表
	MaxSize                  string `field:"maxSize"`                  // 最大尺寸
	SkipCacheControlValues   string `field:"skipCacheControlValues"`   // 忽略的cache-control
	SkipSetCookie            uint8  `field:"skipSetCookie"`            // 是否忽略Set-Cookie Header
	EnableRequestCachePragma uint8  `field:"enableRequestCachePragma"` // 是否支持客户端的Pragma: no-cache
	Cond                     string `field:"cond"`                     // 请求条件
	CreatedAt                uint32 `field:"createdAt"`                // 创建时间
	State                    uint8  `field:"state"`                    // 状态
}

type HTTPCachePolicyOperator struct {
	Id                       interface{} // ID
	AdminId                  interface{} // 管理员ID
	UserId                   interface{} // 用户ID
	IsOn                     interface{} // 是否启用
	Name                     interface{} // 名称
	Key                      interface{} // 缓存Key规则
	Capacity                 interface{} // 容量数据
	Life                     interface{} // 有效期
	Status                   interface{} // HTTP状态码列表
	MaxSize                  interface{} // 最大尺寸
	SkipCacheControlValues   interface{} // 忽略的cache-control
	SkipSetCookie            interface{} // 是否忽略Set-Cookie Header
	EnableRequestCachePragma interface{} // 是否支持客户端的Pragma: no-cache
	Cond                     interface{} // 请求条件
	CreatedAt                interface{} // 创建时间
	State                    interface{} // 状态
}

func NewHTTPCachePolicyOperator() *HTTPCachePolicyOperator {
	return &HTTPCachePolicyOperator{}
}