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
type ExecrscMapper interface {
	RecordRscOp(seqno uint64, rscid int, pid int, rsc string, parameters []string, op string, ret int, tm *time.Timer, count int)
	setRscTerminate(rscid int, op string)
	isRscTerminate(rscid int) bool
	isRscStop(rscid int) bool
	stopTimer(rscid int, op string)
	getCount(rscid int) int 
}

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

//
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
func _NewExecrscMap() ExecrscMapper {
	return ExecrscMap {	
			mapMutex : new(sync.Mutex),
			onRsc : make(map[int]*Execrsc),
		}
}

//
func (e ExecrscMap) RecordRscOp(seqno uint64, rscid int, pid int, rsc string, parameters []string, op string, ret int, tm *time.Timer, count int) {
	_r := _NewExecrsc(seqno, rscid, pid, rsc, parameters, op, ret, tm, count)

	e.mapMutex.Lock()
	defer e.mapMutex.Unlock()

	e.onRsc[rscid] = _r	
}
//
func (e ExecrscMap) setRscTerminate(rscid int, op string) {
	if op == "stop" {
		e.mapMutex.Lock()
		defer e.mapMutex.Unlock()

		if _, _ok := e.onRsc[rscid]; _ok {
			e.onRsc[rscid].terminate = true
		}
	}
}

//
func (e ExecrscMap) isRscTerminate(rscid int) bool {
	if _, _ok := e.onRsc[rscid]; _ok {
		_r := e.onRsc[rscid]
		if _r.terminate {
			return true
		}
	}
	return false
}

//
func (e ExecrscMap) isRscStop(rscid int) bool {
	if _, _ok := e.onRsc[rscid]; _ok {
		_r := e.onRsc[rscid]	
		if _r.op == "stop" {
			return true
		}
	}
	return false
}

//
func (e ExecrscMap) stopTimer(rscid int, op string) {
	if op == "stop"  {
		if _, _ok := e.onRsc[rscid]; _ok {
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
func (e ExecrscMap) getCount(rscid int) int {
	_c := 0
	if _, _ok := e.onRsc[rscid]; _ok {
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
	execrscmap ExecrscMapper
}

//
func (rman *Rmanager) Init(runfn RunFunc, ipcsv ipcs.IpcServer, oprcvch chan interface{}, rscopch chan interface{}, rscmap ExecrscMapper) int {
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
	rman.rscOpRecvMessage_ch = oprcvch
	
	//	
	rman.rscOp_ch = rscopch

	//
	rman.execrscmap = rscmap

	// Start IPCServer
	go rman.ipcServer.Run()

	return consts.CL_OK

}

//
func (rman *Rmanager) Run(list chhandler.ChannelHandler)(int, error) {
	if rman.runFunc != nil {
		rman.runFunc(rman, list)
		return consts.CL_OK, nil
	}
	return consts.CL_ERROR, errors.New("NOT RUNNNING")
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
	_fin_ch := make(chan Execrsc, 12)

	// Make channel for timeout.
	_timeout_ch := make(chan Execrsc, 12)

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
		_args := []string {
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
		_procAttr := os.ProcAttr {
			Files : []*os.File{nil, nil, nil},
			Env   : os.Environ(),
		}

		// Exec RA
		_p, _err := os.StartProcess(_args[0], _args[0:], &_procAttr)
		if _err != nil || _p.Pid < 0 {
                       	debug.DEBUGT.Println("cannot fork child:", rsc, _err, _procAttr.Env)

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
		_s, _err := _p.Wait()
		_tm.Stop()
		
		// OK ? ERROR ?
		_ret := consts.CL_ERROR
		if _s.Exited() && _s.Success() {
			_ret = consts.CL_OK
		}
		
		//
		_cnt := rman.execrscmap.getCount(rscid)
		_fin_ch <- *_NewExecrsc(seqno, rscid, _p.Pid, rsc, parameters, op, _ret, nil, _cnt)
		if _s.Success() == false {
			fmt.Println("---Success() ---> FALSE channel set")
		}
	}()

	// goruoutine for sync/async and manage timeout.
	waitChildFunc := func()(exe Execrsc) {
		var _ri Execrsc
		select {
			case _ri = <- _fin_ch : 
				if  !rman.execrscmap.isRscTerminate(_ri.rscid) {
					fmt.Println("child terminated :", _ri)
					if _ri.op == "monitor" && _ri.count > 0 && _ri.ret == consts.CL_OK && interval > 0 {
						fmt.Println("not send complete monitor to controller", time.Now()) 
					} else {
						select {
							case rman.rscOp_ch <- _ri :
						}
					}
				}
			case _ri = <- _timeout_ch : 
				fmt.Println("timeour occurred.", _ri)
				if _err := syscall.Kill(_ri.pid, syscall.SIGKILL); _err != nil {
					fmt.Println("cannnot child kill:", _err.Error())
				} else {
					fmt.Println("child kill:", _ri.rsc)
				}
				select {
					case rman.rscOp_ch <- _ri :
				}
		}

		// Set of the repetition of the monitor.
		if _ri.ret == consts.CL_OK && interval > 0 {
			rman.setMonitor(seqno, rscid, rsc, parameters, op, interval, timeout, delayMs, async)	
		}	

		rman.execrscmap.setRscTerminate(rscid, op)

		return _ri
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

	return consts.CL_OK
}

//
func NewRmanager(runfn RunFunc, ipcsv ipcs.IpcServer) *Rmanager {
	_cn := new(Rmanager)

	_cn.Init(runfn, 
		ipcsv, 
		make(chan interface{}, 128),
		make(chan interface{}, 128),
		_NewExecrscMap(),
		)

	return _cn
}

//
func _isRmanager(ci interface{}) *Rmanager {
	switch _rman := ci.(type) {
	case *Rmanager:
		return _rman
	default:
	}
	return nil
}
