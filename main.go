package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"webssh/controller"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

//go:embed web/dist/*
var f embed.FS

var (
	port       = flag.Int("p", 5032, "服务运行端口")
	basePathF  = flag.String("b", "", "服务根路径前缀, 如 /webssh")
	v          = flag.Bool("v", false, "显示版本号")
	authInfo   = flag.String("a", "", "开启账号密码登录验证, '-a user:pass'的格式传参")
	timeout    int
	savePass   bool
	basePath   string
	version    string
	buildDate  string
	goVersion  string
	gitVersion string
	username   string
	password   string
)

var sessionToken = func() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("generate session token: %w", err))
	}
	return hex.EncodeToString(b)
}()

var sensitiveLogQueryKeys = map[string]struct{}{
	"auth":          {},
	"authorization": {},
	"pass":          {},
	"password":      {},
	"sessionid":     {},
	"sshinfo":       {},
	"token":         {},
}

func normalizeBasePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	return "/" + strings.Trim(trimmed, "/")
}

func joinBasePath(prefix, path string) string {
	normalizedPrefix := normalizeBasePath(prefix)
	if path == "" || path == "/" {
		if normalizedPrefix == "" {
			return "/"
		}
		return normalizedPrefix + "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if normalizedPrefix == "" {
		return path
	}
	return normalizedPrefix + path
}

func cookiePath(prefix string) string {
	if normalizeBasePath(prefix) == "" {
		return "/"
	}
	return normalizeBasePath(prefix)
}

var indexHTMLCache []byte

func buildIndexHTML(prefix string) []byte {
	if indexHTMLCache != nil {
		return indexHTMLCache
	}
	indexHTML, err := f.ReadFile("web/dist/index.html")
	if err != nil {
		panic(err)
	}
	indexHTMLCache = []byte(strings.ReplaceAll(string(indexHTML), "__WEBSSH_BASE_PATH__", html.EscapeString(normalizeBasePath(prefix))))
	return indexHTMLCache
}

func isAuthorized(c *gin.Context) bool {
	if password == "" {
		return true
	}
	if token, err := c.Cookie("webssh_token"); err == nil && tokenMatches(token) {
		return true
	}
	if tokenMatches(strings.TrimSpace(c.Query("token"))) {
		return true
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = strings.TrimSpace(authHeader[7:])
	}
	return tokenMatches(authHeader)
}

var sessionTokenHash [32]byte

func init() {
	sessionTokenHash = sha256.Sum256([]byte(sessionToken))
}

func tokenMatches(candidate string) bool {
	candidateHash := sha256.Sum256([]byte(candidate))
	return subtle.ConstantTimeCompare(candidateHash[:], sessionTokenHash[:]) == 1
}

func constantTimeStringEqual(candidate, expected string) bool {
	candidateHash := sha256.Sum256([]byte(candidate))
	expectedHash := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(candidateHash[:], expectedHash[:]) == 1
}

func redactRequestPath(path string) string {
	if path == "" {
		return path
	}
	parsed, err := url.ParseRequestURI(path)
	if err == nil {
		parsed.RawQuery = redactQueryString(parsed.RawQuery)
		return parsed.RequestURI()
	}
	prefix, rawQuery, ok := strings.Cut(path, "?")
	if !ok {
		return path
	}
	redactedQuery := redactQueryString(rawQuery)
	if redactedQuery == "" {
		return prefix
	}
	return prefix + "?" + redactedQuery
}

func redactQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "REDACTED_QUERY"
	}
	for key := range values {
		if _, ok := sensitiveLogQueryKeys[strings.ToLower(key)]; ok {
			values[key] = []string{"REDACTED"}
		}
	}
	return values.Encode()
}

func redactingLogFormatter(param gin.LogFormatterParams) string {
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}
	return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n%s",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		param.StatusCode,
		param.Latency,
		param.ClientIP,
		param.Method,
		redactRequestPath(param.Path),
		param.ErrorMessage,
	)
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))) {
	case "https", "wss":
		return true
	default:
		return false
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAuthorized(c) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(401, gin.H{"Msg": "Unauthorized"})
	}
}

func init() {
	flag.IntVar(&timeout, "t", 120, "ssh连接超时时间(min)")
	flag.BoolVar(&savePass, "s", true, "保存ssh密码")
	if envVal, ok := os.LookupEnv("savePass"); ok {
		if b, err := strconv.ParseBool(envVal); err == nil {
			savePass = b
		}
	}
	if envVal, ok := os.LookupEnv("authInfo"); ok {
		*authInfo = envVal
	}
	if envVal, ok := os.LookupEnv("port"); ok {
		if b, err := strconv.Atoi(envVal); err == nil {
			*port = b
		}
	}
	if envVal, ok := os.LookupEnv("basePath"); ok {
		*basePathF = envVal
	} else if envVal, ok := os.LookupEnv("BASE_PATH"); ok {
		*basePathF = envVal
	}
}

func loadRuntimeConfig() bool {
	flag.Parse()
	basePath = normalizeBasePath(*basePathF)
	if *v {
		fmt.Printf("Version: %s\n\n", version)
		fmt.Printf("BuildDate: %s\n\n", buildDate)
		fmt.Printf("GoVersion: %s\n\n", goVersion)
		fmt.Printf("GitVersion: %s\n\n", gitVersion)
		os.Exit(0)
	}
	if *authInfo != "" {
		accountUsername, accountPassword, ok := strings.Cut(*authInfo, ":")
		if !ok || accountUsername == "" || accountPassword == "" {
			fmt.Println("请按'user:pass'的格式来传参或设置环境变量, 且账号密码都不能为空!")
			return false
		}
		username, password = accountUsername, accountPassword
	}
	return true
}

func staticRouter(router *gin.RouterGroup, prefix string) {
	indexHTML := buildIndexHTML(prefix)
	staticPrefix := joinBasePath(prefix, "/static/")
	router.GET("/", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, staticPrefix) {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})
	staticFs, _ := fs.Sub(f, "web/dist/static")
	router.StaticFS("/static", http.FS(staticFs))
}

func newServer() *gin.Engine {
	server := gin.New()
	server.SetTrustedProxies(nil)
	server.Use(gin.LoggerWithFormatter(redactingLogFormatter), gin.Recovery())

	server.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	server.Use(gzip.Gzip(gzip.DefaultCompression))
	if basePath != "" {
		redirectTarget := joinBasePath(basePath, "/")
		redirectToBase := func(c *gin.Context) {
			target := redirectTarget
			if rawQuery := strings.TrimSpace(c.Request.URL.RawQuery); rawQuery != "" {
				target += "?" + rawQuery
			}
			c.Redirect(http.StatusPermanentRedirect, target)
		}
		server.GET(basePath, redirectToBase)
		server.HEAD(basePath, redirectToBase)
	}
	routeGroup := server.Group(basePath)
	staticRouter(routeGroup, basePath)

	apiGroup := routeGroup.Group("/api")
	apiGroup.GET("/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"auth":          password != "",
			"authenticated": isAuthorized(c),
			"savePass":      savePass,
		})
	})

	apiGroup.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"Msg": "invalid request"})
			return
		}
		if constantTimeStringEqual(req.Username, username) && constantTimeStringEqual(req.Password, password) {
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("webssh_token", sessionToken, 3600*24*7, cookiePath(basePath), "", isSecureRequest(c), true)
			c.JSON(200, gin.H{"Msg": "success", "token": sessionToken})
			return
		}
		c.JSON(401, gin.H{"Msg": "invalid credentials"})
	})

	authGroup := routeGroup.Group("")
	authGroup.Use(AuthMiddleware())

	authGroup.GET("/term", func(c *gin.Context) {
		controller.TermWs(c, time.Duration(timeout)*time.Minute)
	})
	file := authGroup.Group("/file")
	{
		file.GET("/list", func(c *gin.Context) {
			c.JSON(200, controller.FileList(c))
		})
		file.GET("/download", func(c *gin.Context) {
			controller.DownloadFile(c)
		})
		file.POST("/upload", func(c *gin.Context) {
			c.JSON(200, controller.UploadFile(c))
		})
		file.GET("/progress", func(c *gin.Context) {
			controller.UploadProgressWs(c)
		})
		file.GET("/content", func(c *gin.Context) {
			c.JSON(200, controller.GetFileContent(c))
		})
		file.POST("/save", func(c *gin.Context) {
			c.JSON(200, controller.SaveFileContent(c))
		})
		file.POST("/create", func(c *gin.Context) {
			c.JSON(200, controller.CreateFile(c))
		})
		file.POST("/mkdir", func(c *gin.Context) {
			c.JSON(200, controller.CreateDirectory(c))
		})
		file.POST("/rename", func(c *gin.Context) {
			c.JSON(200, controller.RenameEntry(c))
		})
	}
	ssh := authGroup.Group("/ssh")
	{
		ssh.POST("/check", func(c *gin.Context) {
			c.JSON(200, controller.CheckSSHConnection(c))
		})
		ssh.POST("/session", func(c *gin.Context) {
			c.JSON(200, controller.CreateSSHSession(c))
		})
		ssh.GET("/saved", func(c *gin.Context) {
			c.JSON(200, controller.ListSavedSSH(c))
		})
		ssh.GET("/system", func(c *gin.Context) {
			c.JSON(200, controller.GetSSHSystemInfo(c))
		})
		ssh.GET("/system/stream", func(c *gin.Context) {
			controller.StreamSSHSystemInfo(c)
		})
		ssh.POST("/saved", func(c *gin.Context) {
			c.JSON(200, controller.SaveSavedSSH(c))
		})
		ssh.POST("/saved/:id/refresh", func(c *gin.Context) {
			c.JSON(200, controller.RefreshSavedSSHInfo(c))
		})
		ssh.DELETE("/saved/:id", func(c *gin.Context) {
			c.JSON(200, controller.DeleteSavedSSH(c))
		})
	}
	return server
}

func main() {
	if !loadRuntimeConfig() {
		return
	}
	server := newServer()
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
	}
}
