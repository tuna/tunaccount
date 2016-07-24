package main

import (
	"reflect"
	"testing"

	"gopkg.in/mgo.v2/bson"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMongo(t *testing.T) {
	Convey("Test MongoDB Find Users", t, func() {
		setDefaultValues(reflect.ValueOf(&dcfg).Elem())
		dcfg.DB.Name = "tunaccount_test"
		initMongo()

		m := getMongo()
		defer m.Close()

		m.UserColl().Insert(
			&User{
				UID:        m.getNextSeq("uid"),
				GID:        m.getNextSeq("gid"),
				Name:       "张三",
				Email:      "zhangsan@example.com",
				Phone:      "+86-010-1234-5678",
				Username:   "zhangsan",
				Password:   generateSSHA("123456"),
				LoginShell: "/bin/bash",
				IsActive:   true,
			},
			&User{
				UID:        m.getNextSeq("uid"),
				GID:        m.getNextSeq("gid"),
				Name:       "李四",
				Email:      "lisi@example.com",
				Phone:      "+86-010-1234-5678",
				Username:   "lisi",
				Password:   generateSSHA("233333"),
				LoginShell: "/bin/bash",
				IsActive:   false,
				Tags:       []string{"testing"},
			},
			&User{
				UID:        m.getNextSeq("uid"),
				GID:        m.getNextSeq("gid"),
				Name:       "王尼玛",
				Email:      "nima@example.com",
				Phone:      "+86-010-1234-5678",
				Username:   "wangnima",
				Password:   generateSSHA("233333"),
				LoginShell: "/bin/bash",
				IsActive:   true,
				Tags:       []string{"testing"},
			},
		)

		Convey("When query all", func() {
			m := getMongo()
			defer m.Close()
			var allUsers []User
			m.UserColl().Find(bson.M{}).All(&allUsers)
			So(len(allUsers), ShouldEqual, 3)
		})

		Convey("When query with filters", func() {
			m := getMongo()
			defer m.Close()
			activeUsers := m.FindUsers(bson.M{}, "")
			So(len(activeUsers), ShouldEqual, 2)
			testUsers := m.FindUsers(bson.M{}, "testing")
			So(len(testUsers), ShouldEqual, 1)
			invalidUsers := m.FindUsers(bson.M{}, "ooxx")
			So(len(invalidUsers), ShouldEqual, 0)
		})

		Reset(func() {
			m := getMongo()
			defer m.Close()
			m.session.DB(m.dbname).DropDatabase()
		})
	})

}
