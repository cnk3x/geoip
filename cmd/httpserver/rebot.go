package main

import (
	"strings"

	"shu.run/log"
	"shu.run/lu"
)

//NoRebot 过滤搜索引擎
func NoRebot() func(h lu.HandlerFunc) lu.HandlerFunc {
	return func(h lu.HandlerFunc) lu.HandlerFunc {
		return func(c lu.Context) error {
			userAgent := strings.ToLower(c.Request().Header.Get("User-Agent"))
			if strings.Contains(userAgent, "spider") || strings.Contains(userAgent, "bot") {
				log.Debug(userAgent, "droped.")
				return c.NoContent(404)
			}
			return h(c)
		}
	}
}
