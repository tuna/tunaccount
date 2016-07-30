package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/appleboy/gin-jwt.v2"
	"gopkg.in/mgo.v2/bson"
)

func runHTTPServer(listenAddr string) {
	r := gin.Default()

	jwtMidware := &jwt.GinJWTMiddleware{
		Realm:      "TUNA",
		Key:        []byte(dcfg.HTTP.SecretKey),
		Timeout:    time.Hour,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(username string, password string, c *gin.Context) (string, bool) {
			m := getMongo()
			defer m.Close()
			users := m.FindUsers(bson.M{"username": username}, "")
			if len(users) != 1 {
				return username, false
			}
			user := users[0]
			return username, user.Authenticate(password)
		},
	}

	r.POST("/login", jwtMidware.LoginHandler)
	api := r.Group("/api/v1")
	api.Use(jwtMidware.MiddlewareFunc())
	{
		api.GET("/refresh_token", jwtMidware.RefreshHandler)
	}

	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			logger.Panic(err.Error())
		}
	}()
}
