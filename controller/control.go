package main

import (
	"../base"
	"../chhandler"
	"../consts"
	"../ipcc"
	"../ipcs"
	mes "../message"
	"../udp"
	"log"
	"syscall"
)

//
type Controller interface {
	Init() int
	Run() int
	Terminate() int
	SendUdpMessage(mes string) int
}

//
type RunFunc func(cnt base.Runner, list chhandler.ChannelHandler) int

//
type Controll struct {
	//
	base.BaseControll
	//
	status        consts.StatusId
	udpSend_ch    chan string
	udpRecv_ch    chan interface{}
	ipcSrvRecv_ch chan interface{}
	ipcClient_ch  chan interface{}
	//
	runFunc       RunFunc
	udpController udp.UdpController
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
	udpc udp.UdpController,
	ipcsv ipcs.IpcServer) int {
	//
	cnt.status = consts.STARTUP
	// Make Chanel
	cnt.InitBase(syscall.SIGTERM, syscall.SIGCHLD)

	// Get map(for clients)
	cnt.clients = ipcsv.GetClientMap()

	// Set MainRun Fun
	cnt.runFunc = runfn

	// Set Udp Controller
	if udpc != nil {
		cnt.udpController = udpc
		cnt.udpSend_ch, cnt.udpRecv_ch = udpc.GetUdpChannel()
	}

	// Set IpcServer Controller
	cnt.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	cnt.ipcServer = ipcsv

	//Start Udp and IPCServer
	if udpc != nil {
		go cnt.udpController.Run(cnt.udpSend_ch, cnt.udpRecv_ch)
	}
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
	if cnt.udpSend_ch != nil {
		close(cnt.udpSend_ch)
	}
	if cnt.udpRecv_ch != nil {
		close(cnt.udpRecv_ch)
	}
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
func (cnt *Controll) SendUdpMessage(mes string) int {
	if cnt.udpSend_ch != nil {
		cnt.udpSend_ch <- mes
	}
	return 0
}

//
func NewControll(
	runfn RunFunc,
	udpc udp.UdpController,
	ipcsv ipcs.IpcServer) *Controll {
	//
	_cn := new(Controll)
	_cn.Init(runfn, udpc, ipcsv)
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
