package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/appleboy/gin-jwt.v2"
	"gopkg.in/mgo.v2/bson"
)

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
		Authenticator: func(username string, password string, c *gin.Context) (string, bool) {
			if rootPwd != "" {
				if username == "root" && rootPwd == password {
					c.Set("user", rootUser)
					return username, true
				}
			}

			m := getMongo()
			defer m.Close()
			users := m.FindUsers(bson.M{"username": username}, "")
			if len(users) != 1 {
				return username, false
			}
			user := users[0]
			if !user.Authenticate(password) {
				return "", false
			}
			c.Set("user", user)
			return username, true
		},
		Authorizator: func(username string, c *gin.Context) bool {
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
