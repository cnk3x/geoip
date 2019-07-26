package main

import (
	"fmt"
	"html/template"
	"os"
	"runtime"
	"strings"

	"shu.run/log"
	"shu.run/lu"

	"shu.run/geoip"
)

var geo = geoip.New("geo_db")

func main() {
	err := geo.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer geo.Close()

	l := lu.New()

	// l.Pre(middleware.AddTrailingSlash())
	l.Pre(NoRebot())
	l.Use(RecoverMw(1))
	l.Use(NewContextMw())

	l.HTTPErrorHandler = httpErrorHandler

	l.Renderer = &iprTemplate{templates: template.Must(template.New("html").Parse(html))}

	l.Any("/:ip", handleSearch)
	l.Any("/", handleSearch)
	l.Any("/version", handleVersion)
	l.Any("/version.*", handleVersion)
	l.Any("/languages", handleLang)
	l.Any("/languages.*", handleLang)
	l.Any("/ping", handlePing)
	l.GET("/favicon.ico", lu.NotFoundHandler)

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "12345"
	}
	log.Info("启动API -> :" + port)
	err = l.Start(":" + port)

	if err != nil {
		log.Fatal(err)
	}
}

func handleSearch(ctx lu.Context) error {
	c := ctx.(*Context)

	ipString := c.GetParam("ip")
	log.Debug("ipString:", ipString)
	if ipString == "" {
		ipString = c.RealIP()
	}

	for _, allow := range allowFmts {
		if strings.HasSuffix(ipString, "."+allow) {
			ipString = strings.TrimSuffix(ipString, "."+allow)
		}
	}

	lang := c.GetParam("lang", "ln", "l")
	if lang == "" {
		languages := strings.Split(c.Request().Header.Get("Accept-Language"), ",")
		if len(languages) > 0 {
			lang = strings.Split(languages[0], ";")[0]
		}
	}

	result, err := geo.Find(ipString, lang)
	if err != nil {
		return c.Output(&geoip.GeoResult{Code: "error", Msg: err.Error()})
	}
	return c.Output(result)
}

func handleUpdate(ctx lu.Context) error {
	errs := make(chan error, 1)
	geo.Update(errs)
	err := <-errs
	if err != nil {
		return ctx.String(200, err.Error())
	}
	return ctx.String(200, "任务已提交")
}

func handleLang(ctx lu.Context) error {
	return ctx.(*Context).Output(geo.Languages())
}

func handleVersion(ctx lu.Context) error {
	return ctx.(*Context).Output(geo.DatabaseVersion())
}

func handlePing(ctx lu.Context) error {
	return ctx.String(200, "pong")
}

//RecoverMw RecoverMw
func RecoverMw(stackSize int64) lu.MiddlewareFunc {
	return func(next lu.HandlerFunc) lu.HandlerFunc {
		return func(c lu.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%v", r)
					}
					if stackSize != -1 {
						size := stackSize
						if size <= 1 {
							size = 4 << 10
						}
						stack := make([]byte, size)
						length := runtime.Stack(stack, stackSize == 1)
						log.Errorf("[PANIC RECOVER] %v %s\n", err, stack[:length])
					} else {
						log.Errorf("[PANIC RECOVER] %v\n", err)
					}
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}
