package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"go.shu.run/lu"
)

var allowFmts = []string{"json", "xml", "html", "text", "txt"}
var contextPool = &sync.Pool{
	New: func() interface{} {
		return &Context{}
	},
}

//NewContextMw NewContextMw
func NewContextMw() lu.MiddlewareFunc {
	return func(h lu.HandlerFunc) lu.HandlerFunc {
		return func(ctx lu.Context) error {
			c := contextPool.Get().(*Context)
			defer contextPool.Put(c)
			c.Context = ctx

			p := strings.TrimSuffix(c.Request().URL.Path, "/")

			format := c.GetParam("fmt", "format", "output")

			for _, ex := range allowFmts {
				if strings.HasSuffix(p, "."+ex) {
					if format == "" {
						format = ex
					}
					p = strings.TrimSuffix(p, "."+ex)
				}
			}

			c.Request().URL.Path = p

			allow := false
			for _, ex := range allowFmts {
				if ex == format {
					allow = true
				}
			}

			if !allow {
				accept := c.Request().Header.Get("Accept")
				switch {
				case strings.Contains(accept, "text/html"):
					format = "html"
				case strings.Contains(accept, "application/json"):
					format = "json"
				case strings.Contains(accept, "application/xml"):
					format = "xml"
				default:
					format = "text"
				}
			}

			c.outputFormat = format

			return h(c)
		}
	}
}

//Context Context
type Context struct {
	lu.Context
	outputFormat string
}

//GetParam GetParam
func (c *Context) GetParam(names ...string) string {
	var v string
	for _, name := range names {
		v = c.FormValue(name)
		if v == "" {
			v = c.Param(name)
		}
		if v == "" {
			v = c.HeaderGet(name)
		}
		if v != "" {
			return v
		}
	}
	return ""
}

//HeaderGet HeaderGet
func (c *Context) HeaderGet(names ...string) string {
	header := c.Request().Header
	var v string
	for _, name := range names {
		if v = header.Get(name); v != "" {
			return v
		}
	}
	return v
}

//Output Output
func (c *Context) Output(data interface{}) error {
	_, pretty := c.QueryParams()["pretty"]
	if !pretty {
		pretty, _ = strconv.ParseBool(c.GetParam("pretty"))
	}

	switch c.outputFormat {
	case "xml":
		if pretty {
			return c.XMLPretty(200, data, "  ")
		}
		return c.XML(200, data)
	case "json":
		if pretty {
			return c.JSONPretty(200, data, "  ")
		}
		return c.JSON(200, data)
	case "html":
		c.ViewModel(data)
		if err := c.Render(200, "html"); err != nil {
			return c.HTML(200, fmt.Sprint(data))
		}
		return nil
	default:
		return c.String(200, fmt.Sprint(data))
	}
}

func httpErrorHandler(err error, c lu.Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*lu.HTTPError); ok {
		code = he.Code
		msg = he.Message
		if he.Internal != nil {
			err = fmt.Errorf("%v, %v", err, he.Internal)
		}
	} else {
		msg = err.Error()
	}

	if _, ok := msg.(string); ok {
		msg = lu.Map{"message": msg}
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			c.NoContent(code)
		} else {
			c.JSON(code, msg)
		}
	}
}
