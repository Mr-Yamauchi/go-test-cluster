// Control project main.go
package main

import (
	"../chhandler"
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
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
}

//
func _processRun(ct *Rmanager, chData chhandler.ChannelHandler) (wait int) {
	
	//
	go func() {
		for {
			ct.Lock()
			for idx := 0; idx < chData.GetLen(); idx ++ {
				select {
				case _sig_ch := <-ct.GetSignalChannel() :
					fmt.Println("SIGNALED")
					switch _sig_ch {
					case syscall.SIGTERM:
						ct.GetExitChannel() <- 1
					case syscall.SIGCHLD:
						fmt.Println("CHILD EXIT")
					default:
						ct.GetExitChannel() <- 1
					}
				case _ch := <- chData.GetCh(idx) :
					chData.Exec(idx, ct, _ch)
				default:
				}
			}
			ct.Unlock()
		}
	}()
	wait = <-ct.Exit_ch

	return
}

//
func _messageHelloHandler(ci interface{}, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	if ct := _isRmanager(ci); ct != nil { 
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
}

//
func _messageResourceHandler(ci interface{}, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	if ct := _isRmanager(ci); ct != nil {
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
}

//
func _processIpcSrvMessage(ci interface{}, data interface{}) {
	//
	if ct := _isRmanager(ci); ct != nil {
		var _ipcTypeMessageFunc = []*ipcs.IpcTypeMessageHandler {
        		{Types: mes.MESSAGE_ID_HELLO, Handler: _messageHelloHandler},
			{Types: mes.MESSAGE_ID_RESOUCE, Handler: _messageResourceHandler},
		}

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

	// Set Channel Handler
	chhandler.ChannelList = chhandler.SetChannelHandler( chhandler.ChannelList, _cn,
				  chhandler.New( _cn.ipcSrvRecv_ch, _processIpcSrvMessage ))
		
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
	_cn.Run(chhandler.ChannelList)

	// Finish
	_terminate(_cn)
}
