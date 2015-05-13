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
	SendIpcToClient(clients map[int]*ClientConnect, dest int, data []byte) int
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
		if _, err := _to.Con.Write(data); err != nil {
			log.Println(err.Error())
		}
	} else {
		log.Println("cannot find client.")
	}

	return 0
}

//
func (ipcs *IpcServerController) clientReceive(c net.Conn) {
	for {
		_buf := make([]byte, consts.BUFF_MAX)
		//
		_nr, err := c.Read(_buf)
		if err != nil {
			log.Println("read error(client disconnected):", err)
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
	_nl, err := net.Listen("unix", ipcs.sockFiles)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	for {
		_con, err := _nl.Accept()
		if err != nil {
			log.Println("accept error:", err)
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
