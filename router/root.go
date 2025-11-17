package router

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Port         any      `toml:"port"`
	Origins      []string `toml:"origins"`
	AllowHeaders []string `toml:"allow_headers"`
	IsProduction bool     `toml:"is_production"`
	Limit        int64    `toml:"limit"` // limit size of request data (megabytes)
}

type Router struct {
	config   *Config
	engine   *gin.Engine
	srv      *http.Server
	basePath string
	limit    int64
	log      *zap.Logger
	lvl      zapcore.Level

	isProduction bool
}

func New(config *Config, basePath string, swagger ...bool) *Router {
	if config == nil {
		config = &Config{
			Port:         8080,
			Origins:      []string{"*"},
			IsProduction: false,
			Limit:        10,
		}
	}

	gin.SetMode(gin.ReleaseMode)

	var (
		log *zap.Logger
	)

	if config.IsProduction {
		pConfig := zap.NewProductionConfig()
		pConfig.Sampling = nil
		log, _ = pConfig.Build()
	} else {
		log, _ = zap.NewDevelopment()
	}

	r := &Router{
		config:   config,
		engine:   gin.New(),
		basePath: setSlashPrefix(basePath),
		limit:    config.Limit,
		log:      log,
		lvl:      zapcore.InfoLevel,

		isProduction: config.IsProduction,
	}
	r.log.Info("Set http request size limit", zap.Int64("limit", r.limit))

	allowOrigins := func() []string {
		if len(config.Origins) == 0 {
			return []string{"*"}
		} else {
			return config.Origins
		}
	}()

	allowHeaders := func() []string {
		if len(config.AllowHeaders) == 0 {
			return []string{"ORIGIN", "Content-Length", "Content-Type", "Access-Control-Allow-Headers", "Access-Control-Allow-Origin", "Authorization", "X-Requested-With", "expires"}
		} else {
			return append(config.AllowHeaders, "ORIGIN", "Content-Length", "Content-Type", "Access-Control-Allow-Headers", "Access-Control-Allow-Origin", "Authorization", "X-Requested-With", "expires")
		}
	}()

	r.engine.Use(gin.Recovery(), r.limitRequestSize(), r.logger)
	r.engine.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     allowHeaders,
		ExposeHeaders:    allowHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.engine.GET("/api/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, nil)
	})
	r.log.Debug("Success to register api", zap.String("method", GET.String()), zap.String("path", "/api/health"))

	r.engine.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, nil)
	})
	r.log.Debug("Success to register api", zap.String("method", GET.String()), zap.String("path", "/health"))

	// swagger API 선언
	if len(swagger) > 0 && swagger[0] {
		r.RegisterHandler("/", GET, func(ctx *gin.Context) {
			ctx.Redirect(http.StatusFound, r.basePath+"/swagger/index.html")
		})
		r.RegisterHandler("/swagger/*any", GET, ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	r.srv = &http.Server{
		Handler:      r.engine,
		Addr:         fmt.Sprintf(":%v", config.Port),
		WriteTimeout: timeout,
		ReadTimeout:  timeout,
	}

	return r
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) SetMiddleware(middleware ...gin.HandlerFunc) {
	r.engine.Use(middleware...)
}

func (r *Router) Run() error {
	go func() {
		if err := r.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			r.log.Panic("Failed to load router", zap.Error(err))
		}
	}()

	r.log.Info("Run", zap.Any("port", r.config.Port))
	return nil
}

func (r *Router) RunTLS(cert, key string) error {
	go func() {
		if err := r.srv.ListenAndServeTLS(cert, key); err != nil && !errors.Is(err, http.ErrServerClosed) {
			r.log.Panic("Failed to load router", zap.Error(err))
		}
	}()

	r.log.Info("RunTLS", zap.Any("port", r.config.Port))
	return nil
}

func (r *Router) Shutdown() error {
	if err := r.srv.Shutdown(context.Background()); err != nil {
		r.log.Error("Shutdown http server", zap.Error(err))
		return err
	}

	r.log.Info("Shutdown", zap.Any("port", r.config.Port))

	return nil
}

func (r *Router) RegisterHandler(path string, method Method, handlers ...gin.HandlerFunc) {
	path = strings.Join([]string{r.basePath, setSlashPrefix(path)}, "")
	switch method {
	case GET:
		r.engine.GET(path, handlers...)
	case POST:
		r.engine.POST(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
	case PUT:
		r.engine.PUT(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
	case DELETE:
		r.engine.DELETE(path, handlers...)
	case PATCH:
		r.engine.PATCH(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
	case WS:
		r.engine.GET(path, append(gin.HandlersChain{connectWS}, handlers...)...)
	default:
		r.log.Panic("Not supported rest method", zap.String("method", method.String()))
	}
	r.log.Debug("Success to register api", zap.String("method", method.String()), zap.String("path", path))
}

func (r *Router) RegisterHandlersToGroup(groupPath string, handlersInfo map[string]map[Method][]gin.HandlerFunc, middleware ...gin.HandlerFunc) {
	group := r.engine.Group(strings.Join([]string{r.basePath, setSlashPrefix(groupPath)}, ""), middleware...)
	for path, innerMap := range handlersInfo {
		for method, handlers := range innerMap {
			switch method {
			case GET:
				group.GET(path, handlers...)
			case POST:
				group.POST(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
			case PUT:
				group.PUT(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
			case DELETE:
				group.DELETE(path, handlers...)
			case PATCH:
				group.PATCH(path, append(gin.HandlersChain{r.getBodyReusable}, handlers...)...)
			case WS:
				group.GET(path, append(gin.HandlersChain{connectWS}, handlers...)...)
			default:
				r.log.Panic("Not supported rest method", zap.String("method", method.String()))
			}
			r.log.Debug("Success to register api", zap.String("method", method.String()), zap.String("path", strings.Join([]string{setSlashPrefix(group.BasePath()), setSlashPrefix(path)}, "")))
		}
	}
}

func (r *Router) RespOk(ctx *gin.Context, res any) {
	ctx.JSON(http.StatusOK, Response[any]{
		Code:    http.StatusOK,
		Message: http.StatusText(http.StatusOK),
		Data:    res,
	})
}

func (r *Router) RespErr(ctx *gin.Context, code int, err error) {
	ctx.JSON(code, Response[any]{
		Code:    code,
		Message: http.StatusText(code),
		Data:    err.Error(),
	})
}

func (r *Router) RespErrWithStatusOK(ctx *gin.Context, code int, err error, message ...string) {
	ctx.JSON(http.StatusOK, Response[any]{
		Code:    code,
		Message: strings.Join(append([]string{http.StatusText(code)}, message...), ". "),
		Data:    err.Error(),
	})
}

func (r *Router) limitRequestSize() gin.HandlerFunc {
	if r.limit == 0 {
		r.limit = 10
	}

	return func(c *gin.Context) {
		maxSize := r.limit * 1024 * 1024
		// 1. First Content-Length header check (fast blocking)
		if contentLength := c.GetHeader("Content-Length"); contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				if size > maxSize {
					r.log.Warn("Request size limit exceeded via Content-Length header",
						zap.Int64("size", size),
						zap.Int64("limit_mb", r.limit),
						zap.String("ip", c.ClientIP()),
						zap.String("path", c.Request.URL.Path))
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{
						"error":          "Request too large",
						"max_size":       fmt.Sprintf("%dMB", r.limit),
						"requested_size": fmt.Sprintf("%.2f MB", float64(size)/(1024*1024)),
						"blocked":        "Request size limit exceeded",
					})
					c.Abort()
					return
				}
			}
		}
		// 2. Set actual body size limit (Content-Length manipulation and chunked request prevention)
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		}
		c.Next()
	}
}

func setSlashPrefix(path string) string {
	if strings.EqualFold(path, "") || strings.HasPrefix(path, "/") {
		return path
	} else {
		return strings.Join([]string{"/", path}, "")
	}
}
