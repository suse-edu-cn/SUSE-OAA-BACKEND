package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// parseQueryBool 解析查询参数中的可选布尔值。
// 规则：
// 1. 参数未传、传空字符串（如 ?key=）、传 "null"、"undefined"、"all" 时返回 (nil, nil)（表示不设过滤/全查）；
// 2. 传 "true"、"1" 返回 (&true, nil)；
// 3. 传 "false"、"0" 返回 (&false, nil)；
// 4. 其他非法字符返回 (_, error)。
func parseQueryBool(c *gin.Context, key string) (*bool, error) {
	raw, exists := c.GetQuery(key)
	if !exists {
		return nil, nil
	}
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "undefined" || trimmed == "all" {
		return nil, nil
	}
	b, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s 参数不合法", key)
	}
	return &b, nil
}
