// Control project main.go
package main

import (
	"../debug"
	"../base"
	"../chhandler"
	"../ipcs"
	"syscall"
	"errors"
	"os"
	"fmt"
	"time"
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
func (rman *Rmanager) ExecRscOp( rsc string, op string, timeout int, delayMs int64) chan int {

	_v := make(chan int)	
	// 
	defer func() {
		// Make channel for Sync RA Prooess. 
		_c := make(chan int)

		// Make channel for timeout.
		_t := make(chan int)

		// Delay.
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		// Start RA Process.
		go func() {
                	var procAttr os.ProcAttr
			args := []string {
				rsc,
				op,
				"",
			}

			// Set OCF_ Parameter.
			os.Setenv("OCF_ROOT", "/usr/lib/ocf")

			// Set Process Attributes.
                	procAttr.Files = []*os.File{nil, nil, nil}
			procAttr.Env = os.Environ()

			// Exec RA
			_p, err := os.StartProcess(args[0], args[0:], &procAttr)
			if err != nil || _p.Pid < 0 {
                        	debug.DEBUGT.Println("cannot fork child:", rsc, err, procAttr.Env)
				_c <- -1
                	} else {
                        	debug.DEBUGT.Println("child start : path:%s pid:%d\n", rsc, _p.Pid)
                	}

			// timeout ...
			fmt.Println("timeout set :", timeout, time.Now())
			_tm := time.AfterFunc(time.Duration(timeout) * time.Millisecond, 
				func() {
					fmt.Println("timeout occrur!!!", time.Now())
					_t <- 1
				} )
							
			// Wait.....
			_s, err := _p.Wait()
			_tm.Stop()
			debug.DEBUGT.Println("WAIT END : child", rsc)
			debug.DEBUGT.Println("WAIT END : p", _p)
			fmt.Println("WAIT END : child", rsc, time.Now())
			if _s.Exited() {
				_c <- 0
			} else {
				_c <- -1
			}
		}()

		// goruoutine for sync/async and manage timeout.
		go func(){
			for {
				select {
					case _v := <-_c : 
						fmt.Println("child terminated.", _v)
						break
					case <-_t : 
						fmt.Println("timeour occurred.")
						break

				}
			}
		}()
	}()

	//	
	return _v
}

//
func (rman *Rmanager) Terminate() int {
	close(rman.ipcSrvRecv_ch)

	rman.TerminateBase()
	debug.DEBUGT.Println("Terminated...")

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
