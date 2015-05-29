// Control project main.go
package main

import (
	"../base"
	"../chhandler"
	"../ipcs"
	"log"
	"syscall"
	"errors"
	"os"
	"fmt"
)

//
type Rmanagers interface {
	Init() int
	Run() int
	Terminate() int
}

//
type RunFunc func(rman base.Runner, list chhandler.ChannelHandler) int

//
type Rmanager struct {
	//
	base.BaseControll
	//
	ipcSrvRecv_ch chan interface{}
	//
	ipcServer ipcs.IpcServer
	//
	runFunc RunFunc
	//
	rscOp_ch chan interface{}
	//
	clients map[int]*ipcs.ClientConnect
}

//
func (rman *Rmanager) Init(runfn RunFunc, ipcsv ipcs.IpcServer) int {
	// Make Chanel
	rman.InitBase(syscall.SIGTERM, syscall.SIGCHLD)
	// Status channel Not Used.
	close(rman.Status_ch)
	rman.Status_ch = nil

	// Get map(for clients)
	rman.clients = ipcsv.GetClientMap()
	// Set MainRun func
	rman.runFunc = runfn
	//
	rman.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	rman.ipcServer = ipcsv

	//
	rman.rscOp_ch = make(chan interface{}, 128)
	// Start IPCServer
	go rman.ipcServer.Run()

	return 0

}

//
func (rman *Rmanager) Run(list chhandler.ChannelHandler)(int, error) {
	if rman.runFunc != nil {
		rman.runFunc(rman, list)
		return 0, nil
	}
	return 1, errors.New("NOT RUNNNING")
}

//
func (rman *Rmanager) ExecRscOp( rsc string, op string, timeout int, delay int) chan int {

	_v := make(chan int)	
	// 
	defer func() {
		_c := make(chan int)

		go func() {
                	var procAttr os.ProcAttr
			args := []string { op, }

			os.Setenv("OCF_ROOT","/usr/lib/ocf")
                	procAttr.Files = []*os.File{nil, nil, nil}
			procAttr.Env = os.Environ()
			_p, err := os.StartProcess(rsc, args, &procAttr)
			if err != nil || _p.Pid < 0 {
                        	log.Printf("cannnot fork child :  path[%s]", rsc, err)
                        	fmt.Printf("cannnot fork child :  path[%s]", rsc, err)
				fmt.Println("ENV print: ",procAttr.Env)
				_c <- -1
                	} else {
                        	log.Printf("child start : path[%s] pid[%d]", rsc, _p.Pid)
                        	fmt.Printf("child start : path[%s] pid[%d]", rsc, _p.Pid)
                	}
			_, err = _p.Wait()
	//		_s, err := _p.Wait()
	//		status := _s.Sys().(syscall.WaitStatus)
	//		_c <- status.ExitStatus()
			_c <- 0
		}()
	}()

	//	
	return _v
}

//
func (rman *Rmanager) Terminate() int {
	close(rman.ipcSrvRecv_ch)

	rman.TerminateBase()
	log.Println("Terminated...")

	return 0
}

//
func NewRmanager(runfn RunFunc, ipcsv ipcs.IpcServer) *Rmanager {
	_cn := new(Rmanager)

	_cn.Init(runfn, ipcsv)

	return _cn
}

//
func _isRmanager(ci interface{}) *Rmanager {
	switch rman := ci.(type) {
	case *Rmanager:
		return rman
	default:
	}
	return nil
}
