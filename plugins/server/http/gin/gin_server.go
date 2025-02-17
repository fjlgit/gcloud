package gin

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/skirrund/gcloud/logger"
	"github.com/skirrund/gcloud/plugins/server/http/gin/prometheus"
	"github.com/skirrund/gcloud/plugins/server/http/gin/zipkin"
	"github.com/skirrund/gcloud/response"
	"github.com/skirrund/gcloud/server"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

type Server struct {
	Srv     *gin.Engine
	Options server.Options
}

const MAX_PRINT_BODY_LEN = 1024

type bodyLogWriter struct {
	gin.ResponseWriter
	bodyBuf *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	//memory copy here!
	w.bodyBuf.Write(b)
	return w.ResponseWriter.Write(b)
}

func NewServer(options server.Options, routerProvider func(engine *gin.Engine)) server.Server {
	srv := &Server{}
	srv.Options = options
	gin.SetMode(gin.ReleaseMode)
	s := gin.New()
	s.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("[GIN] recover:", recovered)

		c.JSON(200, response.Fail(fmt.Sprintf("%v", recovered)))
		return
		//		c.AbortWithStatus(http.StatusInternalServerError)
	}))
	s.Use(cors)
	s.Use(loggingMiddleware)
	zipkin.InitZipkinTracer(s)
	gp := prometheus.New(s)
	s.Use(gp.Middleware())
	// metrics采样
	s.GET("/metrics", gin.WrapH(promhttp.Handler()))

	pprof.Register(s)
	routerProvider(s)
	srv.Srv = s
	return srv
}

func cors(c *gin.Context) {
	method := c.Request.Method
	//origin := c.Request.Header.Get("Origin")
	// //接收客户端发送的origin （重要！）
	// header := c.Writer.Header()
	// header.Set("Access-Control-Allow-Origin", "*")
	// //服务器支持的所有跨域请求的方法
	// header.Set("Access-Control-Allow-Methods", "*")
	// //允许跨域设置可以返回其他子段，可以自定义字段
	// c.Header("Access-Control-Allow-Headers", "*")
	// // 允许浏览器（客户端）可以解析的头部 （重要）
	// //c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
	// //设置缓存时间
	// //	c.Header("Access-Control-Max-Age", "172800")
	// //允许客户端传递校验信息比如 cookie (重要)
	// c.Header("Access-Control-Allow-Credentials", "true")
	if method == "OPTIONS" {
		c.AbortWithStatus(http.StatusNoContent)
	} else {
		c.Next()
	}
}

func loggingMiddleware(ctx *gin.Context) {
	start := time.Now()
	blw := bodyLogWriter{bodyBuf: bytes.NewBufferString(""), ResponseWriter: ctx.Writer}
	ctx.Writer = blw
	ctx.Next()
	strBody := strings.Trim(blw.bodyBuf.String(), "\n")
	if len(strBody) > MAX_PRINT_BODY_LEN {
		strBody = strBody[:(MAX_PRINT_BODY_LEN - 1)]
	}
	defer requestEnd(ctx, start, strBody)
}

func requestEnd(ctx *gin.Context, start time.Time, strBody string) {
	req := ctx.Request
	uri, _ := url.QueryUnescape(req.RequestURI)
	if strings.HasPrefix(uri, "/metrics") {
		strBody = "ignore..."
	}
	logger.Info("\n [GIN] uri:" + uri + "\n [GIN] method:" + req.Method + "\n [GIN] response:" + strBody + "\n [GIN] cost:" + strconv.FormatInt(time.Since(start).Milliseconds(), 10) + "ms")
}

func (server *Server) Shutdown() {
	defer zipkin.Close()
}

func (server *Server) Run(graceful func()) {
	srv := &http.Server{
		Addr:         server.Options.Address,
		Handler:      server.Srv,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	go func() {
		logger.Info("[GIN] server starting on:", server.Options.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Panic("[GIN] listen:", err.Error())
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("[GIN]Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		grace(server, graceful)
		logger.Panic("[GIN]Server forced to shutdown:", err)
	}
	grace(server, graceful)
	logger.Info("[GIN]server has been shutdown")
}

func grace(server *Server, g func()) {
	server.Shutdown()
	g()
}
