package ipcc

import (
	consts "../consts"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

//
type IpcClient interface {
	Run(sch chan string, rch chan string)
}
type IpcClientController struct {
	sockFiles  string
	conn       net.Conn
	ipcrecv_ch chan string
}

//
func (ic *IpcClientController) reader(r io.Reader, ch chan string) {
	_buf := make([]byte, consts.BUFF_MAX)
	for {
		n, err := r.Read(_buf[:])
		if err != nil {
			return
		}
		ch <- string(_buf[0:n])
	}
}

//
func (ic *IpcClientController) TestPrint() {
	fmt.Println("IpcClientController")
}

//
func (ic *IpcClientController) ipcClientStart(ch chan string) {
	_c, err := net.Dial("unix", ic.sockFiles)
	if err != nil {
		panic(err)
	}
	//
	defer func() {
		if err := _c.Close(); err != nil {
			log.Println("close error")
		}
	}()
	//
	go ic.reader(_c, ch)
	//
	for {
		var _msg string = ""
		//
		select {
		case _msg = <-ch:
		default:
			_msg = "hi"
		}
		_, err := _c.Write([]byte(_msg))
		if err != nil {
			log.Println(err)
			//break
		}
		time.Sleep(1e9)
	}
}

//
func (ic *IpcClientController) Connect() net.Conn {
	_c, err := net.Dial("unix", ic.sockFiles)
	if err != nil {
		log.Println(err)
		return nil
	}
	ic.conn = _c
	return _c
}

//
func (ic *IpcClientController) Disconnect() {
	if ic.conn != nil {
		if err := ic.conn.Close(); err != nil {
			log.Println(err)
		}
	}
}

//
func (ic *IpcClientController) SendRecvAsync(msg []byte) int {

	if ic.conn != nil {
		ic.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		_, err := ic.conn.Write(msg)
		if err != nil {
			log.Println(err)
			return 0
		}
	}

	go func() {
		_buf := make([]byte, consts.BUFF_MAX)

		ic.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := ic.conn.Read(_buf[:])
		if err != nil {
			log.Println(err)
			ic.ipcrecv_ch <- ""
		}
		ic.ipcrecv_ch <- string(_buf[0:n])
	}()

	return 0
}

//
func (ic *IpcClientController) SendRecv(msg []byte) string {
	if ic.conn != nil {
		ic.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err := ic.conn.Write(msg)
		if err != nil {
			log.Println(err)
			return ""
		}
		//
		_buf := make([]byte, consts.BUFF_MAX)

		ic.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := ic.conn.Read(_buf[:])
		if err != nil {
			log.Println(err)
			return ""
		}
		//		ic.ipcrecv_ch <- string(_buf[0:n])
		return string(_buf[0:n])
	}
	log.Println("cannnot send : not conneccted")
	return ""
}

//
func (ic *IpcClientController) Run(sch chan string, rch chan string) {
	go ic.ipcClientStart(sch)
}

//
func (ic *IpcClientController) GetReceiveChannel() chan string {
	return ic.ipcrecv_ch
}

//
func New(sf string) *IpcClientController {
	return &IpcClientController{
		sockFiles:  sf,
		ipcrecv_ch: make(chan string),
	}
}
