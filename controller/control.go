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
	clients map[int]*ipcs.ClientConnect
	//rmanager connect info
	rmanConnect *ipcc.IpcClientController
}

//
func (cnt *Controll) Init(
	runfn RunFunc,
	ipcsv ipcs.IpcServer,
	climap  map[int]*ipcs.ClientConnect, 
	rmanc *ipcc.IpcClientController) int {

	//
	cnt.nodeid = 0
	//
	cnt.status = consts.STARTUP
	// Make Chanel
	cnt.InitBase(syscall.SIGTERM, syscall.SIGCHLD)

	// Get map(for clients)
	cnt.clients = climap

	// Set MainRun Fun
	cnt.runFunc = runfn

	// Set IpcServer Controller
	cnt.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	cnt.ipcServer = ipcsv

	//
	cnt.rmanConnect = rmanc

	go cnt.ipcServer.Run()

	return consts.CL_OK

}

//
func (cnt *Controll) Run(list chhandler.ChannelHandler) int {
	if cnt.runFunc != nil {
		cnt.runFunc(cnt, list)
	}
	return consts.CL_OK
}

//
func (cnt *Controll) Terminate() int {
	close(cnt.ipcSrvRecv_ch)

	cnt.TerminateBase()

	log.Println("Terminated...")

	return consts.CL_OK
}


var step int = 1
//
func (cnt *Controll) _resourceControl() int {
	//
	var send bool = true
	var _request mes.MessageResourceControllRequest

	// Todo : Resource placement processing is necessary(Recipe Processor).

	switch step {
		//
		case 1 :
		//
		_request = mes.MessageResourceControllRequest{
		 mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		mes.MessageResourceControllRequestBody {
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
		},
		}
		//
		case 2 :
		//
		_request = mes.MessageResourceControllRequest{
		 mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		mes.MessageResourceControllRequestBody {
		Operation:     "monitor",
		Rscid : 1,
		Resource_Name: "/usr/lib/ocf/resource.d/heartbeat/Dummy2",
		Interval : 5000,
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
		},
		}
/*
		//
		case 3 : 
		//
		_request = mes.MessageResourceControllRequest{
		mes.MessageHeader{
			SeqNo:		 cnt.rmanConnect.GetSeqno(),
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		mes.MessageResourceControllRequestBody {
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
		},
		}
*/
		default : 
			send = false
	}
	//
	if send {
		//cnt.rmanConnect.SendRecvAsync(mes.MakeMessage(_request))
		cnt.rmanConnect.SendRecv(mes.MakeMessage(_request), 20000, _request.Header.SeqNo)
		step = step + 1
		return 1
	} else {
		// None Operation.
		return consts.CL_OK
	}
}

//
func NewControll(
	runfn RunFunc,
	ipcsv ipcs.IpcServer,
	rmanc *ipcc.IpcClientController) *Controll {
	//
	_cn := new(Controll)
	_cn.Init(runfn, ipcsv, ipcsv.GetClientMap(), rmanc)
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
