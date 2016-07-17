package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConfig(t *testing.T) {

	Convey("When using empty config", t, func() {
		cfg, err := loadDaemonConfig("/path/some/thing")
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)
		So(cfg.DB.Name, ShouldEqual, "tunaccount")
	})

}
