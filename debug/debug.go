package debug

import (
	"../envs"
	"fmt"
)

type DEBUGS bool
var DEBUGT = DEBUGS(envs.DEBUG)

func (dbg DEBUGS)Println(a ...interface{}) {
	if (dbg) {
		fmt.Println(a)
	}
}

func (dbg DEBUGS)Printf(format string, a ...interface{}) {
	if (dbg) {
		fmt.Printf(format, a)
	}
}

