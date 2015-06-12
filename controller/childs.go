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
	Start() error
	Stop(os.Signal) error
}

type FuncStart func(chl *ChildControll) error
type FuncStop func(chl *ChildControll, sig syscall.Signal) error

type ChildControll struct {
	childs    []ChildInfo
	funcStart FuncStart
	funcStop  FuncStop
}

//
func (chl *ChildControll) Start() error {
	return chl.funcStart(chl)
}

//
func (chl *ChildControll) Stop(sig syscall.Signal) error {
	return chl.funcStop(chl, sig)
}

//
func NewChildControll(chinfo []ChildInfo, fnStart FuncStart, fnStop FuncStop) *ChildControll {
	_cc := new(ChildControll)

	_cc.childs = chinfo
	_cc.funcStart = _childStart
	_cc.funcStop = _childStop
	if fnStart != nil {
		_cc.funcStart = fnStart
	}
	if fnStop != nil {
		_cc.funcStop = fnStop
	}

	return _cc
}

//
func _childStart(ct *ChildControll) error {
	log.Println("child start")
	//
	for i := 0; i < len(ct.childs); i++ {
		var _procAttr os.ProcAttr
		_procAttr.Files = []*os.File{nil, nil, nil}
		//
		if _p, _err := os.StartProcess(ct.childs[i].path, []string { "start" }, &_procAttr); _err != nil || _p.Pid < 0 {
			log.Printf("cannnot fork child :  path[%s]", ct.childs[i].path)
			return _err
		} else {
			log.Printf("child start : path[%s] pid[%d]", ct.childs[i].path, _p.Pid)
			ct.childs[i].pid = _p.Pid
		}
		if _p, _err := os.FindProcess(ct.childs[i].pid); _p == nil || _err != nil {
			log.Printf("cannnot start child :  path[%s]", ct.childs[i].path)
			return _err
		}
	}
	return nil
}

//
func _childKill(ct *ChildControll) error {
	return _childStop(ct, syscall.SIGKILL)
}

//
func _childStop(ct *ChildControll, sig syscall.Signal) error {
	log.Println("child stop")
	//
	for i := 0; i < len(ct.childs); i++ {
		if ct.childs[i].pid != 0 {
			//
			if _p, _err := os.FindProcess(ct.childs[i].pid); _p != nil {
				//
				if _err := syscall.Kill(ct.childs[i].pid, sig); _err != nil {
					log.Println("child cannot stop:" + _err.Error())
				} else {
					log.Printf("child stop : path[%s] pid[%d]", ct.childs[i].path, ct.childs[i].pid)
				}
				ct.childs[i].pid = 0
			} else if _err != nil {
				return _err
			} else {
				log.Printf("child:path[%s] pid[%d] is already dieing.", ct.childs[i].path, ct.childs[i].pid)
				ct.childs[i].pid = 0
			}
		}
	}
	return nil
}
