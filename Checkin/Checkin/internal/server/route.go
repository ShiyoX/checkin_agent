package server

import (
	"Checkin/internal/handler/agent"
	"Checkin/internal/handler/auth"
	"Checkin/internal/handler/checkin"
	"Checkin/internal/handler/points"
	"Checkin/internal/handler/user"
	"Checkin/internal/middleware"
	"net/http"

	"Checkin/internal/handler/calc"
	"Checkin/pkg/logging"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func SetupRoutes(cfg *viper.Viper) *gin.Engine {
	agent.InitAgentService(cfg)
	r := gin.New()
	r.Use(logging.GinLogger(), logging.GinRecovery(true)) // 日志中间件，记录请求日志
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowHeaders = append(corsCfg.AllowHeaders, "Authorization")
	corsCfg.AllowAllOrigins = true
	r.Use(cors.New(corsCfg)) // CORS 跨域中间件，简单粗暴，直接放行所有跨域请求
	apiV1 := r.Group("/api/v1")
	{
		apiV1.GET("/add", calc.AddHandler())
		apiV1.POST("/users", user.CreateHandler)
		apiV1.POST("/auth/login", auth.LoginHandler) //用户
		apiV1.POST("/auth/refresh", auth.RefreshHandler)
		//注册登录接口后，用户需要携带token访问用户信息接口，所以需要在用户信息接口前面加上认证中间件
		apiV1.Use(middleware.Auth())
		apiV1.GET("/users/me", user.ProfileHandler) //用户信息

		//Checkin Api Group
		checkinGroup := apiV1.Group("/checkins")
		{
			checkinGroup.POST("", checkin.DaylyHandler)
			checkinGroup.GET("/calendar", checkin.CalendarHandler)
			checkinGroup.POST("retroactive", checkin.RetroactiveHandler)
		}

		//points group
		pointsGroup := apiV1.Group("/points")
		{
			pointsGroup.GET("/summary", points.SummaryHandler)
			pointsGroup.GET("/records", points.RecordsHandler)

		}

		//agent group
		agentGroup := apiV1.Group("/agent")
		{
			agentGroup.POST("/chat", agent.ChatHandler)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"msg": "404",
		})
	})
	return r
}
