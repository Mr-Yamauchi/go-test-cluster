// Control project main.go
package main

import (
	"log"
	"os"
	"syscall"
)

//
type ChildInfo struct {
	name string
	path string
	pid  int
}

//
type ChildController interface {
	Start()
	Stop()
}

type FuncStart func(ct *ChildControll)
type FuncStop func(ct *ChildControll)

type ChildControll struct {
	childs    []ChildInfo
	funcStart FuncStart
	funcStop  FuncStop
}

//
func (ch *ChildControll) Start() {
	ch.funcStart(ch)
}

//
func (ch *ChildControll) Stop() {
	ch.funcStop(ch)
}

//
func NewChildControll(chinfo []ChildInfo, fnStart FuncStart, fnStop FuncStop) *ChildControll {
	cc := new(ChildControll)

	cc.funcStart = fnStart
	cc.funcStop = fnStop
	cc.childs = chinfo

	return cc
}

//
func _childStart(ct *ChildControll) {
	log.Println("child start")
	//
	for i := 0; i < len(ct.childs); i++ {
		if _id, err := syscall.ForkExec(ct.childs[i].path, nil, nil); err != nil || _id < 0 {
			log.Printf("cannnot fork child :  path[%s]", ct.childs[i].path)
			panic(err)
		} else {
			log.Printf("child start : path[%s] pid[%d]", ct.childs[i].path, _id)
			ct.childs[i].pid = _id
		}
		if _p, err := os.FindProcess(ct.childs[i].pid); _p == nil || err != nil {
			log.Printf("cannnot start child :  path[%s]", ct.childs[i].path)
			panic(err)
		}
	}
}
//
func _childStop(ct *ChildControll) {
	log.Println("child stop")
	//
	for i := 0; i < len(ct.childs); i++ {
		if ct.childs[i].pid != 0 {
			//
			if _p, err := os.FindProcess(ct.childs[i].pid); _p != nil {
				//
				if err := syscall.Kill(ct.childs[i].pid, syscall.SIGTERM); err != nil {
					log.Println("Child cannot stop:" + err.Error())
				} else {
					log.Printf("child stop : path[%s] pid[%d]", ct.childs[i].path, ct.childs[i].pid)
				}
				ct.childs[i].pid = 0
			} else if err != nil {
				log.Printf(err.Error())
			} else {
				log.Printf("child:path[%s] pid[%d] is dieing.", ct.childs[i].path, ct.childs[i].pid)
			}
		}
	}
}
