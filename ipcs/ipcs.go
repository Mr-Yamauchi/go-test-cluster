package ipcs

import (
	consts "../consts"
	mes "../message"
	"log"
	"net"
	"os"
)

//
type IpcServer interface {
	GetClientMap() map[int]*ClientConnect
	GetRecvChannel() chan interface{}
	SendIpcToClient(map[int]*ClientConnect, int, []byte) int
	Run()
}

type IpcServerController struct {
	sockFiles  string
	ipcrecv_ch chan interface{}
}

type IpcTypeMessageHandler struct {
	Types   int
	Handler func(interface{}, *ClientConnect, []byte, mes.MessageCommon)
}

//
type ClientConnect struct {
	Con     net.Conn
	Message []byte
}

//
func (ipcs IpcServerController) GetClientMap() map[int]*ClientConnect {
	return make(map[int]*ClientConnect)
}

//
func (ipcs IpcServerController) SendIpcToClient(clients map[int]*ClientConnect, dest int, data []byte) int {
	if _to, ok := clients[dest]; ok {
		if _, _err := _to.Con.Write(data); _err != nil {
			log.Println(_err.Error())
			_to.Con.Close()
			delete(clients, dest)
		}
	} else {
		log.Println("cannot find client.")
	}

	return consts.CL_OK
}

//
func (ipcs *IpcServerController) clientReceive(c net.Conn) {
	for {
		_buf := make([]byte, consts.BUFF_MAX)
		//
		_nr, _err := c.Read(_buf);
		if _err != nil {
			log.Println("read error(client disconnected):", _err)
			ipcs.ipcrecv_ch <- "exit"
			c.Close()
			return
		}
		//
		log.Println("client connected")
		_data := _buf[0:_nr]
		//
		_client := &ClientConnect{
			Con:     c,
			Message: _data,
		}
		ipcs.ipcrecv_ch <- _client
	}
}

//
func (ipcs *IpcServerController) ipcServerStart() {
	_nl, _err := net.Listen("unix", ipcs.sockFiles)
	if _err != nil {
		log.Fatal("listen error:", _err)
	}

	for {
		_con, _err := _nl.Accept()
		if _err != nil {
			log.Println("accept error:", _err)
			continue
		}
		go ipcs.clientReceive(_con)
	}
}

//
func (ipcs *IpcServerController) GetRecvChannel() chan interface{} {
	return ipcs.ipcrecv_ch
}

//
func (ipcs *IpcServerController) Run() {
	os.Remove(ipcs.sockFiles)
	go ipcs.ipcServerStart()
}

//
func New(sf string) *IpcServerController {
	return &IpcServerController{
		sockFiles:  sf,
		ipcrecv_ch: make(chan interface{}),
	}
}
