// Control project main.go
package main

import (
	"../debug"
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
)

//
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
}

//
func _messageHelloHandler(ci interface{}, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	if _ct := _isRmanager(ci); _ct != nil {
		var _ms mes.MessageHello

		//Recv HelloMessage Unmarshal
		if _err := json.Unmarshal(recv_mes, &_ms); _err != nil {
			log.Println("Unmarshal ERROR" + _err.Error())
			return
		}

		//Add clients map
		if _, _ok := _ct.clients[_ms.Header.Source_id]; _ok {
			//Already connect(this means reconnet client)
			fmt.Printf("already Connect:%d - map replace\n", _ms.Header.Source_id)
			_ct.clients[_ms.Header.Source_id] = client
			fmt.Printf("len : %d\n", len(_ct.clients))
		} else {
			_ct.clients[_ms.Header.Source_id] = client
		}

		//Hello Response Send to Client
		_response := mes.MessageHello{
			mes.MessageHeader{
				SeqNo : head.Header.SeqNo,
				Destination_id: head.Header.Source_id,
				Source_id:      int(consts.CONTROLLER_ID),
				Types:          int(mes.MESSAGE_ID_HELLO),
			},
			mes.MessageHelloBody {
				Pid:     os.Getpid(),
				Message: "HELLO",
			},
		}

		//
		_ct.ipcServer.SendIpcToClient(_ct.clients, head.Header.Source_id, mes.MakeMessage(_response))
		//
	}
}

//
func _messageResourceHandler(ci interface{}, client *ipcs.ClientConnect, recv_mes []byte, head mes.MessageCommon) {
	if _ct := _isRmanager(ci); _ct != nil {
		var _ms mes.MessageResourceControllRequest

		//Recv MessageResourceControll Unmarshal
		if _err := json.Unmarshal(recv_mes, &_ms); _err != nil {
			fmt.Println("Unmarshal ERROR" + _err.Error())
			return
		}
		fmt.Println(_ms)

		//
		_ct.rscOpRecvMessage_ch <- _ms
	}

}
//
func _processRscOpMessage(ci interface{}, data interface{}) {
	//
	if _ct := _isRmanager(ci); _ct != nil {
		switch _v := data.(type) {
			case mes.MessageResourceControllRequest : 
				_ct.ExecRscOp(_v.Header.SeqNo,
					 _v.Rscid, _v.Resource_Name,
					 mes.ParametersToString(_v.ParamLen, _v.Parameters), _v.Operation, _v.Interval, _v.Timeout, _v.Delay, _v.Async)
			default : 
				debug.DEBUGT.Println("unknown message receive")
			
		}
	}
}

//
func _processRscOpEvent(ci interface{}, data interface{}) {
	fmt.Println("_processRscOpEvent call")
	if _ct := _isRmanager(ci); _ct != nil {
		switch _v := data.(type) {
			case Execrsc :
				fmt.Println("_processRscOpEvent : ", _v)
                		//MessageResourceControll Response Send to Client
		                _response := mes.MessageResourceControllResponse {
                        		mes.MessageHeader{
						SeqNo : _v.seqno,
                                		Destination_id: int(consts.CONTROLLER_ID),
                                		Source_id:      int(consts.RMANAGER_ID),
                                		Types:          int(mes.MESSAGE_ID_RESOUCE_RESPONSE),
                        		},
					mes.MessageResourceControllResponseBody {
                        			Pid:     os.Getpid(),
						Rscid:   _v.rscid,
						Operation: _v.op,
						Resource_Name : _v.rsc,
						ResultCode : _v.ret,
                        			Message: "OK",
					},
                		}
                		//
		                //
		                _ct.ipcServer.SendIpcToClient(_ct.clients, int(consts.CONTROLLER_ID), mes.MakeMessage(_response))
                //
		default:
			log.Println("unknown Event RECV(default)")
			
		}
	}

}

//
func _processIpcSrvMessage(ci interface{}, data interface{}) {
	//
	if _ct := _isRmanager(ci); _ct != nil {
		var _ipcTypeMessageFunc = []*ipcs.IpcTypeMessageHandler{
			{Types: mes.MESSAGE_ID_HELLO, Handler: _messageHelloHandler},
			{Types: mes.MESSAGE_ID_RESOUCE, Handler: _messageResourceHandler},
		}

		switch _v := data.(type) {
		case *ipcs.ClientConnect:
			fmt.Println("RECV(ClientConnect) : " + string(_v.Message))
			//
			_recv_mes := _v.Message
			var _head mes.MessageCommon
			if _err := json.Unmarshal(_recv_mes, &_head); _err != nil {
				fmt.Println("unmarshal ERROR" + _err.Error())
			}
			//
			var _processed bool = false
			for i := 0; i < len(_ipcTypeMessageFunc); i++ {
				if _ipcTypeMessageFunc[i].Types == _head.Header.Types {
					_ipcTypeMessageFunc[i].Handler(_ct, _v, _recv_mes, _head)
					_processed = true
					break
				}
			}
			if _processed == false {
				log.Println("receive Unkown MessageTypes")
			}
		case string:
			if _v == "exit" {
				debug.DEBUGT.Println(_v)
			}
		default:
			log.Println("unknown Data RECV(default)")
		}
	}
}


//
func _initialize() *Rmanager {

	// Setting logging
	_logger, _err := syslog.New(consts.Logpriority, consts.Logtag)
	errs.CheckErrorPanic(_err, "syslog.New Error")
	log.SetOutput(_logger)

	// Mkdir 
	if _err = os.Mkdir("/var/run/resource-agents/", 0777); _err != nil {
		log.Println(_err)
	}

	// Load Configuration
	_config := configure.New("../configure/config.json")
	_config.DumpConfig()

	// Create NewRmanager and Get IpcServer Channel.
	_cn := NewRmanager(
		//
		chhandler.ProcessRun,
		//
		ipcs.New("/tmp/rmanager.sock"),
	)

	// Set IpcServer Message channel handler.
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.ipcSrvRecv_ch, _processIpcSrvMessage))

	// Set RscOp Message channel handler.
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.rscOpRecvMessage_ch, _processRscOpMessage))

	// set RscOp Event channel handler.
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.rscOp_ch, _processRscOpEvent))


	return _cn
}

//
func _terminate(cn *Rmanager) {
	cn.Terminate()
}

//
func main() {
	debug.DEBUGT.Println("START")
	// Init
	_cn := _initialize()

	// Main Loop Running
	_cn.Run(chhandler.ChannelList)

	// Finish
	_terminate(_cn)

	debug.DEBUGT.Println("FINISH")

	os.Exit(0)
}
