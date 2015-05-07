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
type IpcServerAndUdp interface {
	Init() int
	Run() int
	Terminate() int
}

//
type Controller interface {
	IpcServerAndUdp
	SendUdpMessage(mes string) int
}

//
type RunFunc func(ct base.Runner, list chhandler.ChannelHandler) int

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
func (ct *Controll) Init(
	runfn RunFunc,
	udpc udp.UdpController,
	ipcsv ipcs.IpcServer) int {
	//
	ct.status = consts.STARTUP
	// Make Chanel
	ct.InitBase(syscall.SIGTERM, syscall.SIGCHLD)

	// Get map(for clients)
	ct.clients = ipcsv.GetClientMap()

	// Set MainRun Fun
	ct.runFunc = runfn

	// Set Udp Controller
	ct.udpController = udpc
	ct.udpSend_ch, ct.udpRecv_ch = udpc.GetUdpChannel()

	// Set IpcServer Controller
	ct.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	ct.ipcServer = ipcsv

	//Start Udp and IPCServer
	go ct.udpController.Run(ct.udpSend_ch, ct.udpRecv_ch)
	go ct.ipcServer.Run()

	return 0

}

//
func (ct *Controll) Run(list chhandler.ChannelHandler) int {
	if ct.runFunc != nil {
		ct.runFunc(ct, list)
	}
	return 0
}

//
func (ct *Controll) Terminate() int {
	close(ct.udpSend_ch)
	close(ct.udpRecv_ch)
	close(ct.ipcSrvRecv_ch)

	ct.TerminateBase()

	log.Println("Terminated...")

	return 0
}

//
func (ct *Controll) _resourceControl() int {
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
	ct.rmanConnect.SendRecvAsync(mes.MakeMessage(_request))
	//
	return 0
}

//
func (ct *Controll) SendUdpMessage(mes string) int {
	ct.udpSend_ch <- mes
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
	switch ct := ci.(type) {
	case *Controll:
		return ct
	default:
	}
	return nil
}
