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
	"sync"
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
type Execrsc struct {
	seqno   uint64
	rscid 	int
	pid 	int
	rsc	string	
	param   []string
	op      string
	ret	int
	tm   	*time.Timer
	count	int
	terminate bool				//Todo : Always false
}

type ExecrscMap struct {
	mapMutex	*sync.Mutex
	onRsc map[int]*Execrsc
}

//
func _NewExecrsc(seqno uint64, rscid int, pid int, rsc string, parameters []string, op string, ret int, tm *time.Timer, count int)(r *Execrsc)  {
        return &Execrsc {
                seqno,
                rscid,
                pid,
                rsc,
                parameters,
                op,
                ret,
                tm,
                count,
                false,
        }
}
//
func _NewExecrscMap() ExecrscMap {
	return ExecrscMap {	
			mapMutex : new(sync.Mutex),
			onRsc : make(map[int]*Execrsc),
		}
}

//
func (e *ExecrscMap) RecordRscOp(seqno uint64, rscid int, pid int, rsc string, parameters []string, op string, ret int, tm *time.Timer, count int) {
	_r := _NewExecrsc(seqno, rscid, pid, rsc, parameters, op, ret, tm, count)

	e.mapMutex.Lock()
	defer e.mapMutex.Unlock()

	e.onRsc[rscid] = _r	
}
//
func (e *ExecrscMap) setRscTerminate(rscid int, op string) {
	if op == "stop" {
		e.mapMutex.Lock()
		defer e.mapMutex.Unlock()

		if _, ok := e.onRsc[rscid]; ok {
			e.onRsc[rscid].terminate = true
		}
	}
}

//
func (e *ExecrscMap) isRscTerminate(rscid int) bool {
	if _, ok := e.onRsc[rscid]; ok {
		_r := e.onRsc[rscid]
		if _r.terminate {
			return true
		}
	}
	return false
}

//
func (e *ExecrscMap) isRscStop(rscid int) bool {
	if _, ok := e.onRsc[rscid]; ok {
		r := e.onRsc[rscid]	
		if r.op == "stop" {
			return true
		}
	}
	return false
}

//
func (e *ExecrscMap) stopTimer(rscid int, op string) {
	if op == "stop"  {
		if _, ok := e.onRsc[rscid]; ok {
			e.mapMutex.Lock()
			defer e.mapMutex.Unlock()

			_r := e.onRsc[rscid]
			if _r.tm != nil {
				_r.tm.Stop()	
			}
		}
	}
}

//
func (e *ExecrscMap) getCount(rscid int) int {
	_c := 0
	if _, ok := e.onRsc[rscid]; ok {
		_c = e.onRsc[rscid].count 
	}
	return _c
}

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
	execrscmap ExecrscMap
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
	rman.execrscmap = _NewExecrscMap()

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
func (rman *Rmanager) setMonitor( seqno uint64, rscid int, rsc string, parameters []string, op string, interval int64, timeout int64, delayMs int64, async bool)(t *time.Timer) {

	rman.execrscmap.RecordRscOp(seqno, rscid, 0, rsc, parameters, op, 0, nil, 1)

	//set Monito by time.AfterFunc.
	_tm := time.AfterFunc(
		time.Duration(interval) * time.Millisecond, 
		func() {
			if !rman.execrscmap.isRscStop(rscid) {
				rman.ExecRscOp(seqno, rscid, rsc, parameters, op, interval, timeout, delayMs, true)
			}
		},
	)

	rman.execrscmap.RecordRscOp(seqno, rscid, 0, rsc, parameters, op, 0, _tm, 1)

	return _tm
}

//
func (rman *Rmanager) ExecRscOp( seqno uint64, rscid int, rsc string, parameters []string, op string, interval int64, timeout int64, delayMs int64, async bool) {

	// Make channel for finish RA Prooess. 
	_c := make(chan Execrsc, 128)

	// Make channel for timeout.
	_timeout_ch := make(chan Execrsc, 128)

	// not monitor add onRsc map.
	if interval == 0 {
		rman.execrscmap.stopTimer(rscid, op)

		rman.execrscmap.RecordRscOp(seqno, rscid, 0, rsc, parameters, op, 0, nil, 0)
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

			_timeout_ch <- *_NewExecrsc(seqno, rscid, _p.Pid, rsc, parameters, op, consts.CL_NOT_FORK, nil, 0)

			return
              	} else {
                       	debug.DEBUGT.Println("child start : path:", rsc, " pid:", _p.Pid)
               	}

		// Control timeout start...
		_tm := time.AfterFunc(
			time.Duration(timeout) * time.Millisecond, 
			func() {
				_timeout_ch <- *_NewExecrsc(seqno, rscid, _p.Pid, rsc, parameters, op, consts.CL_ERROR, nil, 0)
			})
						
		// Wait.....
		_s, err := _p.Wait()
		_tm.Stop()
		
		// OK ? ERROR ?
		ret := consts.CL_ERROR
		if _s.Exited() && _s.Success() {
			ret = consts.CL_NORMAL_END
		}
		
		//
		cnt := rman.execrscmap.getCount(rscid)
		_c <- *_NewExecrsc(seqno, rscid, _p.Pid, rsc, parameters, op, ret, nil, cnt)
		if _s.Success() == false {
			fmt.Println("---Success() ---> FALSE channel set")
		}
	}()

	// goruoutine for sync/async and manage timeout.
	waitChildFunc := func()(exe Execrsc) {
		var r Execrsc
		select {
			case r = <- _c : 
				if  !rman.execrscmap.isRscTerminate(r.rscid) {
					fmt.Println("child terminated :", r)
					if r.op == "monitor" && r.count > 0 && r.ret == consts.CL_NORMAL_END && interval > 0 {
						fmt.Println("not send complete monitor to controller", time.Now()) 
					} else {
						select {
							case rman.rscOp_ch <- r :
						}
					}
				}
			case r = <- _timeout_ch : 
				fmt.Println("timeour occurred.", r)
				if err := syscall.Kill(r.pid, syscall.SIGKILL); err != nil {
					fmt.Println("cannnot child kill:",err.Error())
				} else {
					fmt.Println("child kill:",r.rsc)
				}
				select {
					case rman.rscOp_ch <- r :
				}
		}

		// Set of the repetition of the monitor.
		if r.ret == consts.CL_NORMAL_END && interval > 0 {
			rman.setMonitor(seqno, rscid, rsc, parameters, op, interval, timeout, delayMs, async)	
		}	

		rman.execrscmap.setRscTerminate(rscid, op)

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
