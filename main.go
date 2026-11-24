package main

import (
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/wuonhai/webook/internal/repository"
	"github.com/wuonhai/webook/internal/repository/dao"
	"github.com/wuonhai/webook/internal/service"
	"github.com/wuonhai/webook/internal/web"
	"github.com/wuonhai/webook/internal/web/middleware"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {

	server := initServer()
	db := initDb()
	userHandler := initUser(db)
	userHandler.RegisterRoutes(server)

	server.Run("localhost:8080")
}

func initServer() *gin.Engine {
	server := gin.Default()

	// Add CORS middleware
	server.Use(cors.New(cors.Config{
		// AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:  []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{"x-jwt-token"},
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost")
		},
	}))

	// Add session middleware
	store := cookie.NewStore([]byte("secret"))
	server.Use(sessions.Sessions("webookSSID", store))

	server.Use(middleware.NewLoginMiddlewareBuilder().
		IgnorePaths("/users/login", "/users/signup").
		Build())

	return server
}

func initDb() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:rootpassword@tcp(localhost:3306)/webook"))
	if err != nil {
		// initial database failed, panic the goroutines
		panic(err)
	}

	if err := dao.InitTable(db); err != nil {
		panic(err)
	}

	return db
}

func initUser(db *gorm.DB) *web.UserHandler {

	userDao := dao.NewUserDao(db)
	userRepo := repository.NewUserRepository(userDao)
	userSvc := service.NewUserService(userRepo)
	userHandler := web.NewUserHandler(userSvc)

	return userHandler
}
