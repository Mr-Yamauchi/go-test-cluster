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

//
func (cnt *Controll) _resourceControl() int {
	//
	_request := mes.MessageResourceControllRequest{
		Header: mes.MessageHeader{
			Destination_id: int(consts.RMANAGER_ID),
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE),
		},
		Operation:     "monitor",
		Resource_Name: "Dummy",
		Parameters: []mes.Parameter{
			{
				Name:  "monitor",
				Value: "0",
			},
		},
	}
	//
	cnt.rmanConnect.SendRecvAsync(mes.MakeMessage(_request))
	//
	return 0
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
