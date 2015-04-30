package main

import (
	base "../base"
	"../consts"
	ipcc "../ipcc"
	ipcs "../ipcs"
	mes "../message"
	udp "../udp"
	"log"
)

//
type IpcTypeMessageHandler struct {
	Types   int
	Handler func(*Controll, *ipcs.ClientConnect, []byte, mes.MessageCommon)
}

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
//
type RunFunc func(ct *Controll) int
type ProcessMessageFunc func(ct *Controll, data interface{})
type ProcessStatusFunc func(ct *Controll, data consts.StatusId) int
type ProcessIpcClientMessageFunc func(ct *Controll, data string)

//
type Controll struct {
	//
	base.BaseControll
	//
	status       consts.StatusId
	status_ch    chan consts.StatusId
	udpSend_ch   chan string
	udpRecv_ch   chan string
	ipcSrvRecv_ch   chan interface{}
	ipcClient_ch chan string
	//
	runFunc       RunFunc
	udpController udp.UdpController
	ipcServer     ipcs.IpcServer
	//
	//
	processStatusHander     ProcessStatusFunc
	processUdpMessageHander ProcessMessageFunc
	processIpcSrvMessageHander ProcessMessageFunc
	ipcTypeMessageHandler       []*IpcTypeMessageHandler
	//
	clients map[int]*ipcs.ClientConnect
	//rmanager connect info
	rmanConnect *ipcc.IpcClientController
	processIpcClintMessageHandler ProcessIpcClientMessageFunc
}

//
func (ct *Controll) Init(
	runfn RunFunc,
	udpc udp.UdpController,
	ipcsv ipcs.IpcServer,
	stsfn ProcessStatusFunc,
	udpfn ProcessMessageFunc,
	ipcfn ProcessMessageFunc,
	ipcms []*IpcTypeMessageHandler) int {
	//
	ct.status = consts.STARTUP
	// Make Chanel
	ct.InitBase()

	ct.status_ch = make(chan consts.StatusId, 2)
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

	//Set Status Handler
	ct.processStatusHander = stsfn
	//Set UDP/IPC Message Handler
	ct.processUdpMessageHander = udpfn
	ct.processIpcSrvMessageHander = ipcfn
	ct.ipcTypeMessageHandler = ipcms

	//Start Udp and IPCServer
	go ct.udpController.Run(ct.udpSend_ch, ct.udpRecv_ch)
	go ct.ipcServer.Run()

	return 0

}
//
func (ct *Controll) Run() int {
	if ct.runFunc != nil {
		ct.runFunc(ct)
	}
	return 0
}

//
func (ct *Controll) Terminate() int {
	close(ct.status_ch)
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
	ipcsv ipcs.IpcServer,
	stsfn ProcessStatusFunc,
	udpfn ProcessMessageFunc,
	ipcfn ProcessMessageFunc,
	ipcms []*IpcTypeMessageHandler) *Controll {
	//
	_cn := new(Controll)
	_cn.Init(runfn, udpc, ipcsv, stsfn, udpfn, ipcfn, ipcms)
	//
	return _cn
}
