package main

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"gopkg.in/mgo.v2/bson"

	. "github.com/smartystreets/goconvey/convey"
)

func TestImport(t *testing.T) {
	Convey("Test Import to MongoDB", t, func() {
		setDefaultValues(reflect.ValueOf(&dcfg).Elem())
		dcfg.DB.Name = "tunaccount_test"
		initMongo()

		m := getMongo()
		defer m.Close()

		var dumpContent = `
{
	"users": [
		{
			"uid": 1001,
			"gid": 1000,
			"name": "张三",
			"email": "zhangsan@example.com",
			"username": "zhangsan",
			"login_shell": "/bin/bash",
			"is_active": true
		},
		{
			"uid": 2070,
			"gid": 1000,
			"name": "李四",
			"email": "lisi@example.com",
			"username": "lisi",
			"login_shell": "/bin/bash",
			"is_active": true
		}
	],
	"posix_groups": [
		{
			"gid": 1000,
			"name": "users"
		}
	]
}`

		Convey("import json", func() {
			m := getMongo()
			defer m.Close()
			tmpfile, err := ioutil.TempFile("", "tunasync")
			So(err, ShouldBeNil)
			defer os.Remove(tmpfile.Name())

			err = ioutil.WriteFile(tmpfile.Name(), []byte(dumpContent), 0644)
			So(err, ShouldBeNil)

			var counter mongoCounter
			m.CounterColl().Find(bson.M{"_id": "uid"}).One(&counter)
			So(counter.Seq, ShouldEqual, 2000)

			err = importJSON(tmpfile.Name(), m)
			So(err, ShouldBeNil)

			var allUsers []User
			m.UserColl().Find(bson.M{}).All(&allUsers)
			So(len(allUsers), ShouldEqual, 2)

			m.CounterColl().Find(bson.M{"_id": "uid"}).One(&counter)
			So(counter.Seq, ShouldEqual, 2070)
		})

		Reset(func() {
			m := getMongo()
			defer m.Close()
			m.session.DB(m.dbname).DropDatabase()
		})
	})

}
