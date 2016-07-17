package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSSHA(t *testing.T) {

	Convey("When validating a SSHA", t, func() {
		password := "123456"
		hash := "{SSHA}JW40sBDnfwzy0LYpfcDSe4DC0agVVi+M"
		So(validateSSHA(password, hash), ShouldBeTrue)
	})

	Convey("When generating a SSHA", t, func() {
		password := "123456"
		hash := generateSSHA(password)
		So(validateSSHA(password, hash), ShouldBeTrue)
	})

}
