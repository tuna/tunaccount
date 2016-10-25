package main

import (
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2/bson"

	"github.com/gin-gonic/gin"
)

type apiResp struct {
	Msg string `json:"msg"`
}

type loginResp struct {
	Msg    string `json:"msg"`
	Expire string `json:"expire"`
	Token  string `json:"token"`
}

type passwdForm struct {
	Username string `form:"username" json:"username" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

type userProfileForm struct {
	UID   int    `json:"uid"`
	GID   int    `json:"gid"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`

	Username   string `json:"username"`
	LoginShell string `json:"login_shell"`

	IsActive bool `json:"is_active"`
	IsAdmin  bool `json:"is_admin"`

	Tags []string `json:"tags"`
}

func apiUpdatePassowrd(c *gin.Context) {
	iuser, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"msg": "Login Required"})
		return
	}

	user, ok := iuser.(User)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"msg": "Login Required"})
		return
	}

	var form passwdForm
	if c.BindJSON(&form) != nil {
		if c.Bind(&form) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "Invalid Request"})
			return
		}
	}

	if user.Username != form.Username {
		if !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"msg": "Permission Denied"})
			return
		}
	}

	user.Passwd(form.Password)

	m := getMongo()
	defer m.Close()

	err := m.UserColl().
		Update(
			bson.M{"username": form.Username},
			bson.M{"$set": bson.M{"password": user.Password}},
		)
	if err != nil {
		err = fmt.Errorf("Failed to update password: %s", err.Error())
		logger.Error(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "Password updated"})
}

func apiListUsers(c *gin.Context) {

	m := getMongo()
	defer m.Close()
	users := []userProfileForm{}
	err := m.UserColl().Find(bson.M{}).All(&users)
	if err != nil {
		err = fmt.Errorf("Failed to list users: %s", err.Error())
		logger.Error(err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}
