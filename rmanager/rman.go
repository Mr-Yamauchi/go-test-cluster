// Control project main.go
package main

import (
	base "../base"
	ipcs "../ipcs"
	mes "../message"
	"log"
)

//
type IpcServerAndUdp interface {
	Init() int
	Run() int
	Terminate() int
}

//
type IpcTypeMessageHandler struct {
	Types   int
	Handler func(*Rmanager, *ipcs.ClientConnect, []byte, mes.MessageCommon)
}

//
type RunFunc func(ct *Rmanager) int
type ProcessMessageFunc func(ct *Rmanager, data interface{})

//
type Rmanager struct {
	//
	base.BaseControll
	//
	ipcSrvRecv_ch chan interface{}
	//
	ipcServer ipcs.IpcServer
	//
	runFunc                  RunFunc
	processIpcSrvMessageHandler ProcessMessageFunc
	ipcMessageHandler        []*IpcTypeMessageHandler
	//
	clients map[int]*ipcs.ClientConnect
}

//
func (ct *Rmanager) Init(runfn RunFunc, ipcsv ipcs.IpcServer, ipcfn ProcessMessageFunc, ipcms []*IpcTypeMessageHandler) int {
	// Make Chanel
	ct.InitBase()

	// Get map(for clients)
	ct.clients = ipcsv.GetClientMap()
	// Set MainRun func 
	ct.runFunc = runfn
	//
	ct.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	ct.ipcServer = ipcsv

	// Set IPC Message Handler
	ct.processIpcSrvMessageHandler = ipcfn
	ct.ipcMessageHandler = ipcms

	// Start IPCServer
	go ct.ipcServer.Run()

	return 0

}

//
func (ct *Rmanager) Run() int {
	if ct.runFunc != nil {
		ct.runFunc(ct)
	}
	return 0
}

//
func (ct *Rmanager) Terminate() int {
	close(ct.ipcSrvRecv_ch)

	ct.TerminateBase()
	log.Println("Terminated...")

	return 0
}

//
func NewRmanager(runfn RunFunc, ipcsv ipcs.IpcServer, ipcfn ProcessMessageFunc, ipcms []*IpcTypeMessageHandler) *Rmanager {
	_cn := new(Rmanager)

	_cn.Init(runfn, ipcsv, ipcfn, ipcms)

	return _cn
}
