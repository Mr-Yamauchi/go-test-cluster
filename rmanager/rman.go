// Control project main.go
package main

import (
	"../consts"
	"../debug"
	"../base"
	"../chhandler"
	"../ipcs"
	"syscall"
	"errors"
	"os"
	"fmt"
	"time"
	"strings"
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
	rscOpRecvMessage_ch chan interface{}
	//
	rscOp_ch chan interface{}
	//
	clients map[int]*ipcs.ClientConnect
	//
	onRsc map[int]*Execrsc
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
	rman.rscOpRecvMessage_ch = make(chan interface{}, 128)
	
	//	
	rman.rscOp_ch = make(chan interface{}, 128)

	//
	rman.onRsc = make(map[int]*Execrsc)

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
type Execrsc struct {
	rscid 	int
	pid 	int
	rsc	string	
	param   []string
	op      string
	ret	int
	tm   	*time.Timer
	terminate bool				//Todo : Always false
}
//
func NewExecrsc(rscid int, pid int, rsc string, parameters []string, op string, ret int, tm *time.Timer)(r *Execrsc)  {
	return &Execrsc {
		rscid,
		pid,
		rsc,		
		parameters,
		op,
		ret,
		tm,
		false,
	}
}

//
func (rman *Rmanager) setRscTerminate( rscid int, op string ) {
	if op == "stop" {
		if _, ok := rman.onRsc[rscid]; ok {
			rman.onRsc[rscid].terminate = true
		}
	}
}

//
func (rman *Rmanager) isRscTerminate( rscid int ) bool {
	if _, ok := rman.onRsc[rscid]; ok {
		e := rman.onRsc[rscid]
		if e.terminate {
			return true
		}
	}
	return false
}

//
func (rman *Rmanager) setMonitor( rscid int, rsc string, parameters []string, op string, interval int64, timeout int64, delayMs int64, async bool)(t *time.Timer) {

	rman.onRsc[rscid] = NewExecrsc(rscid, 0, rsc, parameters, op, 0, nil)

	_tm := time.AfterFunc(
		time.Duration(interval) * time.Millisecond, 
		func() {
			if _, ok := rman.onRsc[rscid]; ok {
				r := rman.onRsc[rscid]	
				if r.op != "stop" {
					rman.ExecRscOp(rscid, rsc, parameters, op, interval, timeout, delayMs, true)
				}
			}
		},
	)

	rman.onRsc[rscid] = NewExecrsc(rscid, 0, rsc, parameters, op, 0, _tm)

	return _tm
}

//
func (rman *Rmanager) ExecRscOp( rscid int, rsc string, parameters []string, op string, interval int64, timeout int64, delayMs int64, async bool) {

	// Make channel for finish RA Prooess. 
	_c := make(chan Execrsc, 128)

	// Make channel for timeout.
	_t := make(chan Execrsc, 128)

	// not monitor add onRsc map.
	if interval == 0 {
		if op == "stop"  {
			if _, ok := rman.onRsc[rscid]; ok {
				e := rman.onRsc[rscid]
				if e.tm != nil {
					e.tm.Stop()	
				}
			}

		}
		rman.onRsc[rscid] = NewExecrsc(rscid, 0, rsc, parameters, op, 0, nil)
	}

	// Delay.
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	// Start RA Process.
	go func() {
		args := []string {
			rsc,
			op,
			"",
		}

		// Set OCF_ROOT and OCF_RESKEY_xxx  Parameter.
		os.Setenv("OCF_ROOT", "/usr/lib/ocf")
		for i := 0; i<len(parameters); i++ {
			idx := strings.Index(parameters[i], "=")
			if idx > 0 {
				os.Setenv("OCF_RESKEY_" + parameters[i][:idx], parameters[i][idx+1:])
			}
		}

		// Set Process Attributes.
		procAttr := os.ProcAttr {
			Files : []*os.File{nil, nil, nil},
			Env   : os.Environ(),
		}

		// Exec RA
		_p, err := os.StartProcess(args[0], args[0:], &procAttr)
		if err != nil || _p.Pid < 0 {
                       	debug.DEBUGT.Println("cannot fork child:", rsc, err, procAttr.Env)

			_t <- *NewExecrsc(rscid, _p.Pid, rsc, parameters, op, consts.CL_NOT_FORK, nil)

			return
              	} else {
                       	debug.DEBUGT.Println("child start : path:%s pid:%d\n", rsc, _p.Pid)
               	}

		// Control timeout start...
		_tm := time.AfterFunc(
			time.Duration(timeout) * time.Millisecond, 
			func() {
				_t <- *NewExecrsc(rscid, _p.Pid, rsc, parameters, op, consts.CL_ERROR, nil)
			})
						
		// Wait.....
		_s, err := _p.Wait()
		_tm.Stop()

		//
		ret := consts.CL_ERROR
		if _s.Exited() && _s.Success() {
			ret = consts.CL_NORMAL_END
		}
		_c <- *NewExecrsc(rscid, _p.Pid, rsc, parameters, op, ret, nil)
	}()

	// goruoutine for sync/async and manage timeout.
	waitChildFunc := func()(exe Execrsc) {
		var r Execrsc
		select {
			case r = <-_c : 
				if  !rman.isRscTerminate(r.rscid) {
					fmt.Println("child terminated.", r)
					rman.rscOp_ch <- r
				}
			case r = <-_t : 
				fmt.Println("timeour occurred.", r)
				if err := syscall.Kill(r.pid, syscall.SIGKILL); err != nil {
					fmt.Println("cannnot child kill:",err.Error())
				} else {
					fmt.Println("child kill:",r.rsc)
				}
				rman.rscOp_ch <- r
		}

		// Set of the repetition of the monitor.
		if r.ret == 0 && interval > 0 {
			rman.setMonitor(rscid, rsc, parameters, op, interval, timeout, delayMs, async)	
		}	

		rman.setRscTerminate(rscid, op)

		return r
	}

	// Change async/sync call by goroutine.	
	if (async) {
		go func(){
			waitChildFunc()
		}()
	} else {
		waitChildFunc()
	}		
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
