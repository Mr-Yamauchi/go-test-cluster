// Control project main.go
package main

import (
	"../base"
	"../ipcs"
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
type RunFunc func(ct *Rmanager) int

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
	//
	clients map[int]*ipcs.ClientConnect
}

//
func (ct *Rmanager) Init(runfn RunFunc, ipcsv ipcs.IpcServer) int {
	// Make Chanel
	ct.InitBase(syscall.SIGTERM, syscall.SIGCHLD)

	// Get map(for clients)
	ct.clients = ipcsv.GetClientMap()
	// Set MainRun func 
	ct.runFunc = runfn
	//
	ct.ipcSrvRecv_ch = ipcsv.GetRecvChannel()
	ct.ipcServer = ipcsv

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
func NewRmanager(runfn RunFunc, ipcsv ipcs.IpcServer) *Rmanager {
	_cn := new(Rmanager)

	_cn.Init(runfn, ipcsv)

	return _cn
}
