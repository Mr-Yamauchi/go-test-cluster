// Control project main.go
package main

import (
	configure "../configure"
	consts "../consts"
	"../errs"
	ipcs "../ipcs"
	mes "../message"
	"encoding/json"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime"
	"syscall"
)

//
type IpcTypeMessageHandler struct {
        Types   int        
	Handler func(*Rmanager, *ipcs.ClientConnect, []byte, mes.MessageCommon)
}

type ProcessMessageFunc func(ct *Rmanager, data interface{})

//
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
}

//
func _processRun(ct *Rmanager) (wait int) {
	//
	go func() {
		for {
			select {
			case _sig_ch := <-ct.Signal_ch:
				fmt.Println("SIGNALED")
				switch _sig_ch {
				case syscall.SIGTERM:
					ct.Exit_ch <- 1
				case syscall.SIGCHLD:
					fmt.Println("CHILD EXIT")
				default:
					ct.Exit_ch <- 1
				}
			case _ipc_ch := <-ct.ipcSrvRecv_ch:
				fmt.Println("IPC RECEIVE")
				_processIpcSrvMessage(ct, _ipc_ch)
			default:
			}
		}
	}()
	wait = <-ct.Exit_ch

	return
}

//
func _messageHelloHandler(ct *Rmanager, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	var ms mes.MessageHello
	//Recv HelloMessage Unmarshal
	if err := json.Unmarshal(recv_mes, &ms); err != nil {
		log.Println("Unmarshal ERROR" + err.Error())
		return
	}
	//Add clients map
	if _, ok := ct.clients[ms.Header.Source_id]; ok {
		//Already connect(this means reconnet client)
		fmt.Printf("already Connect:%d - map replace\n", ms.Header.Source_id)
		ct.clients[ms.Header.Source_id] = client
		fmt.Printf("len : %d\n", len(ct.clients))
	} else {
		ct.clients[ms.Header.Source_id] = client
	}
	//Hello Response Send to Client
	_response := mes.MessageHello{
		Header: mes.MessageHeader{
			Destination_id: head.Header.Source_id,
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_HELLO),
		},
		Pid:     os.Getpid(),
		Message: "HELLO",
	}
	//
	ct.ipcServer.SendIpcToClient(ct.clients, head.Header.Source_id, mes.MakeMessage(_response))
	//
}

//
func _messageResourceHandler(ct *Rmanager, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	var ms mes.MessageResourceControllRequest
	//Recv MessageResourceControll Unmarshal
	if err := json.Unmarshal(recv_mes, &ms); err != nil {
		fmt.Println("Unmarshal ERROR" + err.Error())
		return
	}
	fmt.Println(ms)
	//MessageResourceControll Response Send to Client
	_response := mes.MessageHello{
		Header: mes.MessageHeader{
			Destination_id: head.Header.Source_id,
			Source_id:      int(consts.CONTROLLER_ID),
			Types:          int(mes.MESSAGE_ID_RESOUCE_RESPONSE),
		},
		Pid:     os.Getpid(),
		Message: "ACK",
	}
	//
	//
	ct.ipcServer.SendIpcToClient(ct.clients, head.Header.Source_id, mes.MakeMessage(_response))
}

//
var _ipcTypeMessageFunc = []*IpcTypeMessageHandler {
        {Types: mes.MESSAGE_ID_HELLO, Handler: _messageHelloHandler},
	{Types: mes.MESSAGE_ID_RESOUCE, Handler: _messageResourceHandler},
}

//
func _processIpcSrvMessage(ct *Rmanager, data interface{}) {
	//
	switch _v := data.(type) {
	case *ipcs.ClientConnect:
		fmt.Println("RECV(ClientConnect) : " + string(_v.Message))
		//
		_recv_mes := []byte(_v.Message)
		var _head mes.MessageCommon
		if err := json.Unmarshal(_recv_mes, &_head); err != nil {
			fmt.Println("unmarshal ERROR" + err.Error())
		}
		//
		var _processed bool = false
		for i := 0; i < len(_ipcTypeMessageFunc); i++ {
			if _ipcTypeMessageFunc[i].Types == _head.Header.Types {
				_ipcTypeMessageFunc[_head.Header.Types].Handler(ct, _v, _recv_mes, _head)
				_processed = true
				break
			}
		}
		if _processed == false {
			log.Println("receive Unkown MessageTypes")
		}
	case string:
		if _v == "exit" {
			fmt.Println(_v)
		}
	default:
		log.Println("unknown Data RECV(default)")
	}
}

//
func _initialize()*Rmanager {
	// Setting logging
	_logger, err := syslog.New(consts.Logpriority, consts.Logtag)
	errs.CheckErrorPanic(err, "syslog.New Error")
	log.SetOutput(_logger)

	// Load Configuration
	_config := configure.New("../configure/config.json")
	_config.DumpConfig()

	// Create NewRmanager
	_cn := NewRmanager(
		//
		_processRun,
		//
		ipcs.New("/tmp/rmanager.sock"),
	)
	
	return _cn
}
//
func _terminate(cn *Rmanager){
	cn.Terminate()
}
//
func main() {
	// Init
	_cn := _initialize()

	// Main Loop Running
	_cn.Run()

	// Finish
	_terminate(_cn)
}
