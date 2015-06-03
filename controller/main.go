package main

import (
	"../debug"
	"../chhandler"
	configure "../configure"
	"../consts"
	"../errs"
	ipcc "../ipcc"
	ipcs "../ipcs"
	mes "../message"
	"encoding/json"
	"fmt"
	"log"
	"log/syslog"
	"os"
	"runtime"
	"syscall"
	"../corosync"
)

//
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
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
		default:
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
			case mes.MESSAGE_ID_RESOUCE_RESPONSE:
				var ms mes.MessageResourceControllResponse
				if err := json.Unmarshal(_recv_mes, &ms); err != nil {
					log.Println("Unmarshal ERROR" + err.Error())
					return
				}
				fmt.Println("IPC RECEIVE from Server(1) :", data)
				ct.Status_ch <- consts.CONTROL_RESOURCE
			case mes.MESSAGE_ID_HELLO:
				fmt.Println("IPC RECEIVE from Server(2) :", data)
				ct.Status_ch <- consts.CONTROL_RESOURCE
			default:
				fmt.Println("IPC RECEIVE from Server(3) :", data)
			}
		default:
		}
	}
}

//
func _initialize() (*Controll, *ChildControll) {
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
		chhandler.ProcessRun,
		//
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
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.Status_ch, _processStatus))
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.ipcSrvRecv_ch, _processIpcSrvMessage))
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_cn.ipcClient_ch, _processIpcClientMessage))

	return _cn, _ch
}

//
func _processConfchgCallback(ci interface{}, data interface{}) {
	fmt.Println("_processConfchgCallback")
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
			case corosync.CorosyncConfchg :
				fmt.Println("--member_ent :", len(_v.Member_list))
				for i:= 0; i<len(_v.Member_list);i++ {
					fmt.Printf("-- member(%d:%d)\n", _v.Member_list[i].Nodeid, _v.Member_list[i].Pid)
				}
				fmt.Println("--left_ent :", len(_v.Left_list))
				for j:= 0; j<len(_v.Left_list);j++ {
					fmt.Printf("-- left  (%d:%d)\n", _v.Left_list[j].Nodeid, _v.Left_list[j].Pid)
				}
				fmt.Println("--join_ent :", len(_v.Join_list))
				for k:= 0; k<len(_v.Join_list);k++ {
					fmt.Printf("-- joined(%d:%d)\n", _v.Join_list[k].Nodeid, _v.Join_list[k].Pid)
				}
				corosync.SendClusterMessage("xyz----XYZ2")
		}
	}
}

//
func _processMsgDeliverCallback(ci interface{}, data interface{}) {
	fmt.Println("_processMsgDeliverCallback")
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
			case corosync.CorosyncDeliver: 
				fmt.Println("_processMsgDeliverCallback msg:", _v.Msg)
		}
	}
}

//
func _processTotemchgCallback(ci interface{}, data interface{}) {
	fmt.Println("_processTotemchgCallback")
	if ct := _isControll(ci); ct != nil {
		switch _v := data.(type) {
			case corosync.CorosyncTotemchg : 
				fmt.Println("_processTotemchgCallback() len :", len(_v.Member_list))
				for i:= 0; i < len(_v.Member_list); i++ { 
					fmt.Println("active_member : ", _v.Member_list[i])
				}
		}
	}
}

//
func _terminate(cn *Controll, ch *ChildControll) {
	ch.Stop(syscall.SIGTERM)
	cn.Terminate()
}

//
func main() {
	debug.DEBUGT.Println("START")

	// Init
	_cn, _ch := _initialize()

	// corosync connect Init/set ch handler/Run
	_chConfchg, _chMsgDeliv, _chTotemchg := corosync.Init();
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_chConfchg, _processConfchgCallback))
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_chMsgDeliv, _processMsgDeliverCallback))
	chhandler.ChannelList = chhandler.SetChannelHandler(chhandler.ChannelList, _cn,
		chhandler.New(_chTotemchg, _processTotemchgCallback))
	//start poll corosync event.
	corosync.Run();

	// Main Loop Running
	_cn.Run(chhandler.ChannelList)

	// Finish
	_terminate(_cn, _ch)

	debug.DEBUGT.Println("FINISH")
}
