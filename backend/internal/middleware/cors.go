package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 内置允许的来源，部署到新域名时可通过 extraOrigins 追加。
var defaultAllowedOrigins = []string{
	"https://list.tuoxie.asia",
	"http://list.tuoxie.asia",
	"http://localhost:18090",
	"http://127.0.0.1:18090",
}

// CORS 只放行白名单内的 Origin，并处理预检请求。
func CORS(extraOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(defaultAllowedOrigins)+len(extraOrigins))
	for _, origin := range defaultAllowedOrigins {
		allowed[origin] = true
	}
	for _, origin := range extraOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" && allowed[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !allowed[origin] {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
