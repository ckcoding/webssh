package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
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
	v          = flag.Bool("v", false, "显示版本号")
	authInfo   = flag.String("a", "", "开启账号密码登录验证, '-a user:pass'的格式传参")
	timeout    int
	savePass   bool
	version    string
	buildDate  string
	goVersion  string
	gitVersion string
	username   string
	password   string
)

var sessionToken = func() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}()

func isAuthorized(c *gin.Context) bool {
	if password == "" {
		return true
	}
	if token, err := c.Cookie("webssh_token"); err == nil && token == sessionToken {
		return true
	}
	if strings.TrimSpace(c.Query("token")) == sessionToken {
		return true
	}
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		authHeader = strings.TrimSpace(authHeader[7:])
	}
	return authHeader == sessionToken
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
	flag.Parse()
	if *v {
		fmt.Printf("Version: %s\n\n", version)
		fmt.Printf("BuildDate: %s\n\n", buildDate)
		fmt.Printf("GoVersion: %s\n\n", goVersion)
		fmt.Printf("GitVersion: %s\n\n", gitVersion)
		os.Exit(0)
	}
	if *authInfo != "" {
		accountInfo := strings.Split(*authInfo, ":")
		if len(accountInfo) != 2 || accountInfo[0] == "" || accountInfo[1] == "" {
			fmt.Println("请按'user:pass'的格式来传参或设置环境变量, 且账号密码都不能为空!")
			os.Exit(0)
		}
		username, password = accountInfo[0], accountInfo[1]
	}
}

func staticRouter(router *gin.Engine) {
	router.GET("/", func(c *gin.Context) {
		indexHTML, _ := f.ReadFile("web/dist/" + "index.html")
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	})
	staticFs, _ := fs.Sub(f, "web/dist/static")
	router.StaticFS("/static", http.FS(staticFs))
}

func main() {
	server := gin.Default()
	server.SetTrustedProxies(nil)

	server.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	server.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(server)

	server.GET("/api/status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"auth":          password != "",
			"authenticated": isAuthorized(c),
			"savePass":      savePass,
		})
	})

	server.POST("/api/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"Msg": "invalid request"})
			return
		}
		if req.Username == username && req.Password == password {
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("webssh_token", sessionToken, 3600*24*7, "/", "", isSecureRequest(c), true)
			c.JSON(200, gin.H{"Msg": "success", "token": sessionToken})
			return
		}
		c.JSON(401, gin.H{"Msg": "invalid credentials"})
	})

	authGroup := server.Group("")
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
	}
	ssh := authGroup.Group("/ssh")
	{
		ssh.POST("/check", func(c *gin.Context) {
			c.JSON(200, controller.CheckSSHConnection(c))
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
		ssh.DELETE("/saved/:id", func(c *gin.Context) {
			c.JSON(200, controller.DeleteSavedSSH(c))
		})
	}
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
	}
}
