package main

import (
	"../chhandler"
	configure "../configure"
	"../consts"
	"../errs"
	ipcc "../ipcc"
	ipcs "../ipcs"
	mes "../message"
	udp "../udp"
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
func _processRun(ct *Controll, chData []*chhandler.ChannelHandlerData) (wait int) {
	//
	go func() {
		for {
			ct.Lock()
			for idx := 0; idx < len(chData); idx ++ {
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
				case _ch := <- chData[idx].Ch :
					chData[idx].Handler(ct, _ch)
				default :
				}
			}
			ct.Unlock()
		}
	}()
	ct.status_ch <-consts.STARTUP
	wait = <-ct.Exit_ch

	return
}

//
func _messageHelloHandler(ci interface{}, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	
	if ct := _isControll(ci); ct != nil {
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
//
func _processUdpMessage(ci interface{}, data interface{}) {
	//
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
		case string:
			fmt.Println("UDP RECEIVE : " + _v)
		default:
		}
	}
	
		
}

//
func _processIpcSrvMessage(ci interface{}, data interface{}) {
	//
	if ct := _isControll(ci); ct != nil {
		//
		var _ipcTypeMessageFunc = []*ipcs.IpcTypeMessageHandler{
			{Types: mes.MESSAGE_ID_HELLO, Handler: _messageHelloHandler},
		}
		//
		fmt.Println("IPC RECEIVE(from Client)")

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

func _processStatus(ci interface{}, data interface{}) {
	//
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
		case int:
			fmt.Println(consts.StatusId(_v))
			switch _v {
			case consts.STARTUP:
				ct.status = consts.ALL_CLIENT_UP

				/* TESET */
				//Make Hello Request.
				_request := mes.MessageHello{
					Header: mes.MessageHeader{
						Destination_id: int(consts.RMANAGER_ID),
						Source_id:      int(consts.CONTROLLER_ID),
						Types:          int(mes.MESSAGE_ID_HELLO),
					},
					Pid:     os.Getpid(),
					Message: "HELLO",
				}
				//Send Hello Request.
				ct.rmanConnect.SendRecvAsync(mes.MakeMessage(_request))
			case consts.ALL_CLIENT_UP:
			case consts.LOAD_RESOURCE_SETTING:
			case consts.CONTROL_RESOURCE:
				ct._resourceControl()
			case consts.PENDING:
			case consts.OPERATIONAL:
			}
		default : 
		}
	}
}
//
func _processIpcClientMessage(ci interface{}, data interface{}) {
	//
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
		case string:
			_recv_mes := []byte(_v)
			var _head mes.MessageCommon

			if err := json.Unmarshal(_recv_mes, &_head); err != nil {
				fmt.Println("unmarshal ERROR" + err.Error())
			}

			switch _head.Header.Types {
			case mes.MESSAGE_ID_RESOUCE_RESPONSE :
				var ms mes.MessageResourceControllResponse
				if err := json.Unmarshal(_recv_mes, &ms); err != nil {
					log.Println("Unmarshal ERROR" + err.Error())
					return
				}
				fmt.Println("IPC RECEIVE from Server(1) :", data)
			case mes.MESSAGE_ID_HELLO : 
				fmt.Println("IPC RECEIVE from Server(2) :", data)
				ct.status_ch <-consts.CONTROL_RESOURCE
			default:
				fmt.Println("IPC RECEIVE from Server(3) :", data)
			}
		default : 
		}
	}
}
//
func _initialize()(*Controll, *ChildControll) {
	// Setting logging
	_logger, err := syslog.New(consts.Logpriority, consts.Logtag)
	errs.CheckErrorPanic(err, "syslog.New Error")
	log.SetOutput(_logger)

	// Load Configuration
	_config := configure.New("../configure/config.json")
	_config.DumpConfig()

	// Create NewControll
	_cn := NewControll(
		//
		_processRun,
		//
		udp.New(),
		ipcs.New("/tmp/controller.sock"),
	)

	// Rmanager Connect
	_cn.rmanConnect = ipcc.New("/tmp/rmanager.sock")
	_cn.rmanConnect.Connect()
	_cn.ipcClient_ch = _cn.rmanConnect.GetReceiveChannel()

	// Create Child Control 
	_ch := NewChildControll(
		[]ChildInfo{
			{"controller", "/home/yamauchi/test.sh", 0},
			//      {"resource_manager", "../ipc_client/rmanager", 0},
		},
		nil,
		nil,
	)

	// Start Child
	if err2 := _ch.Start(); err2 != nil {
		panic(err2)
	}

	// Set Channel Handler
	chhandler.ChannelList = chhandler.SetChannelHandler( chhandler.ChannelList, _cn,
				 & chhandler.ChannelHandlerData{ Ch : _cn.status_ch, Handler : _processStatus } ) 
	chhandler.ChannelList = chhandler.SetChannelHandler( chhandler.ChannelList, _cn,
				 & chhandler.ChannelHandlerData{ Ch : _cn.ipcSrvRecv_ch, Handler : _processIpcSrvMessage } ) 
	chhandler.ChannelList = chhandler.SetChannelHandler( chhandler.ChannelList, _cn,
				 & chhandler.ChannelHandlerData{ Ch : _cn.ipcClient_ch, Handler : _processIpcClientMessage } ) 
	chhandler.ChannelList = chhandler.SetChannelHandler( chhandler.ChannelList, _cn,
				 & chhandler.ChannelHandlerData{ Ch : _cn.udpRecv_ch,  Handler : _processUdpMessage } ) 

	return _cn, _ch
}
//
func _terminate(cn *Controll, ch *ChildControll){
	ch.Stop(syscall.SIGTERM)	
	cn.Terminate()
}
//
func main() {
	// Init
	_cn, _ch := _initialize()

	//
	_cn.SendUdpMessage(fmt.Sprintf("BBB"))

	// Main Loop Running
	_cn.Run(chhandler.ChannelList)

	// Finish 
	_terminate(_cn, _ch)
}
