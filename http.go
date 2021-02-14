package main

import (
	"errors"
	"net/http"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

type login struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func runHTTPServer(listenAddr, secretKey, rootPwd string) {
	r := gin.Default()
	if rootPwd != "" {
		logger.Warning("Root password is enabled!")
	}

	rootUser := User{
		UID: 0, GID: 0,
		Username: "root",
		Name:     "Root Admin",
		IsActive: true,
		IsAdmin:  true,
	}

	jwtMidware := &jwt.GinJWTMiddleware{
		Realm:      "TUNA",
		Key:        []byte(secretKey),
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginVals login
			if err := c.ShouldBind(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			username := loginVals.Username
			password := loginVals.Password

			if rootPwd != "" {
				if username == "root" && rootPwd == password {
					c.Set("user", rootUser)
					return username, nil
				}
			}

			m := getMongo()
			defer m.Close()
			users := m.FindUsers(bson.M{"username": username}, "")
			if len(users) != 1 {
				return username, errors.New("Wrong user or password")
			}
			user := users[0]
			if !user.Authenticate(password) {
				return "", errors.New("Wrong user or password")
			}
			c.Set("user", user)
			return username, nil
		},
		Authorizator: func(data interface{}, c *gin.Context) bool {
			username, ok := data.(string)
			if !ok {
				return false
			}

			if username == "root" {
				c.Set("user", rootUser)
				return true
			}
			m := getMongo()
			defer m.Close()
			users := m.FindUsers(bson.M{"username": username}, "")
			if len(users) != 1 {
				return false
			}
			user := users[0]
			if !user.IsActive {
				return false
			}
			c.Set("user", user)
			return true
		},
	}

	r.POST("/login", jwtMidware.LoginHandler)
	api := r.Group("/api/v1")
	api.Use(jwtMidware.MiddlewareFunc())
	{
		api.GET("/refresh_token", jwtMidware.RefreshHandler)
		api.POST("/admin/passwd", apiUpdatePassowrd)
		api.GET("/users/", apiListUsers)
	}

	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		logger.Noticef("HTTP Serving on: %s", listenAddr)
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Panic(err.Error())
		}
	}()
}
