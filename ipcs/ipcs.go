package ipcs

import (
	consts "../consts"
	"log"
	"net"
	"os"
)

//
type IpcServer interface {
	GetClientMap()(map[int]*ClientConnect)
	GetRecvChannel() chan interface{}
	SendIpcToClient(clients map[int]*ClientConnect, dest int, data []byte) int
	Run()
}
type IpcServerController struct {
	sockFiles string
	ipcrecv_ch chan interface{}
	
}

//
type ClientConnect struct {
	Con     net.Conn
	Message []byte
}

//
func (is IpcServerController) GetClientMap()(map[int]*ClientConnect) {
	return  make(map[int]*ClientConnect)
}

//
func (is IpcServerController) SendIpcToClient(clients map[int]*ClientConnect, dest int, data []byte) int {
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
func (is *IpcServerController) clientReceive(c net.Conn) {
	for {
		_buf := make([]byte, consts.BUFF_MAX)
		//
		_nr, err := c.Read(_buf)
		if err != nil {
			log.Println("read error(client disconnected):", err)
			is.ipcrecv_ch <- "exit"
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
		is.ipcrecv_ch <- _client
	}
}

//
func (is *IpcServerController) ipcServerStart() {
	_nl, err := net.Listen("unix", is.sockFiles)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	for {
		_con, err := _nl.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		go is.clientReceive(_con)
	}
}
//
func (is *IpcServerController) GetRecvChannel() chan interface{} {
	return is.ipcrecv_ch
}
//
func (is *IpcServerController) Run() {
	os.Remove(is.sockFiles)
	go is.ipcServerStart()
}

//
func New(sf string) *IpcServerController {
	return &IpcServerController {
			sockFiles : sf,
			ipcrecv_ch : make(chan interface{}),
		}
}

