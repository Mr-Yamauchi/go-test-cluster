package main

import (
	"../base"
	"../chhandler"
	"../consts"
	"../ipcc"
	"../ipcs"
	mes "../message"
	"log"
	"syscall"
)

//
type Controller interface {
	Init() int
	Run() int
	Terminate() int
}

//
type RunFunc func(cnt base.Runner, list chhandler.ChannelHandler) int

//
type Controll struct {
	//
	base.BaseControll
	//
	nodeid		uint
	//
	status        consts.StatusId
	ipcSrvRecv_ch chan interface{}
	ipcClient_ch  chan interface{}
	//
	runFunc       RunFunc
	ipcServer     ipcs.IpcServer
	//
	//
	clients map[int]*ipcs.ClientConnect
	//rmanager connect info
	rmanConnect *ipcc.IpcClientController
}

//
func (cnt *Controll) Init(
	runfn RunFunc,
	ipcsv ipcs.IpcServer) int {

	//
	cnt.nodeid = 0
	//
	cnt.status = consts.STARTUP
	// Make Chanel
	cnt.InitBase(syscall.SIGTERM, syscall.SIGCHLD)

	// Get map(for clients)
	cnt.clients = ipcsv.GetClientMap()

	// Set MainRun Fun
	cnt.runFunc = runfn

	// Set IpcServer Controller
	cnt.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	cnt.ipcServer = ipcsv

	go cnt.ipcServer.Run()

	return 0

}

//
func (cnt *Controll) Run(list chhandler.ChannelHandler) int {
	if cnt.runFunc != nil {
		cnt.runFunc(cnt, list)
	}
	return 0
}

//
func (cnt *Controll) Terminate() int {
	close(cnt.ipcSrvRecv_ch)

	cnt.TerminateBase()

	log.Println("Terminated...")

	return 0
}


var step int = 1
//
func (cnt *Controll) _resourceControl() int {
	//
	var send bool = true
	var _request mes.MessageResourceControllRequest

	// Todo : Resource placement processing is necessary

	switch step {
		case 1 :
		_request = mes.MessageResourceControllRequest{
		Header: mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		Operation:     "start",
		Rscid : 1,
		Resource_Name: "/usr/lib/ocf/resource.d/heartbeat/Dummy2",
		Interval : 0,
		Timeout	: 30000,
		Delay : 0,	
		Async : false,
		ParamLen : 1,
		Parameters: []mes.Parameter{
			{
				Name:  "start",
				Value: "0",
			},
		},
		}
		case 2 :
		_request = mes.MessageResourceControllRequest{
		Header: mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		Operation:     "monitor",
		Rscid : 1,
		Resource_Name: "/usr/lib/ocf/resource.d/heartbeat/Dummy2",
		Interval : 10,
		Timeout	: 20000,
		Delay : 0,	
		Async : true,
		ParamLen : 1,
		Parameters: []mes.Parameter{
			{
				Name:  "monitor",
				Value: "0",
			},
		},
		}
/*
		case 3 : 
		_request = mes.MessageResourceControllRequest{
		Header: mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		Operation:     "stop",
		Rscid : 1,
		Resource_Name: "/usr/lib/ocf/resource.d/heartbeat/Dummy2",
		Interval : 0,
		Timeout	: 20,
		Delay : 0,	
		Async : false,
		ParamLen : 1,
		Parameters: []mes.Parameter{
			{
				Name:  "monitor",
				Value: "0",
			},
		},
		}
*/
		default : 
			send = false
	}
	//
	if send {
		//cnt.rmanConnect.SendRecvAsync2(mes.MakeMessage(_request))
		cnt.rmanConnect.SendRecv2(mes.MakeMessage(_request), 20000, _request.Header.SeqNo)
		step = step + 1
		return 1
	} else {
		// None Operation.
		return 0
	}
}

//
func NewControll(
	runfn RunFunc,
	ipcsv ipcs.IpcServer) *Controll {
	//
	_cn := new(Controll)
	_cn.Init(runfn, ipcsv)
	//
	return _cn
}

//
func _isControll(ci interface{}) *Controll {
	switch cnt := ci.(type) {
	case *Controll:
		return cnt
	default:
	}
	return nil
}
