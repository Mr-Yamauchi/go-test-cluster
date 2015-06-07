package ipcc

import (
	consts "../consts"
	"io"
	"log"
	"net"
)

//
/*
type IpcClient interface {
	Run(sch chan string, rch chan string)
}
*/

type IpcClientController struct {
	sockFiles  string
	conn       	net.Conn
	ipcrecv_ch 	chan interface{}
	send_ch 	chan []byte
	read_ch 	chan string
	write_ch 	chan []byte
}

//
func (ipcc *IpcClientController) reader(r io.Reader, ch chan string) {
	_buf := make([]byte, consts.BUFF_MAX)
	for {
		n, err := r.Read(_buf[:])
		if err != nil {
			return
		}
		select {
			case ch <- string(_buf[0:n]):
		}
	}
}

//
func (ipcc *IpcClientController) writer(c net.Conn, ch chan []byte ) {
	for {
		var _msg []byte
		//
		select {
			case _msg = <- ch:
		}
		_, err := c.Write(_msg)
		if err != nil {
			log.Println(err)
		}
	}
}

//
/*
func (ipcc *IpcClientController) TestPrint() {
	fmt.Println("IpcClientController")
}
*/

//
/*
func (ipcc *IpcClientController) ipcClientStart(ch chan string) {
	_c, err := net.Dial("unix", ipcc.sockFiles)
	if err != nil {
		panic(err)
	}
	//
	defer func() {
		if err := _c.Close(); err != nil {
			log.Println("close error")
		}
	}()

	// Start reader.
	go ipcc.reader(_c, ch)
	
	// Start writer loop.
	for {
		var _msg string = ""
		//
		select {
			case _msg = <- ch:
			//default:
			//	_msg = "hi"
		}
		_, err := _c.Write([]byte(_msg))
		if err != nil {
			log.Println(err)
			//break
		}
		//time.Sleep(1e9)
	}
}
*/

//
func (ipcc *IpcClientController) Connect() net.Conn {
	_c, err := net.Dial("unix", ipcc.sockFiles)
	if err != nil {
		log.Println(err)
		return nil
	}
	ipcc.conn = _c
	return _c
}

//
func (ipcc *IpcClientController) Disconnect() {
	if ipcc.conn != nil {
		if err := ipcc.conn.Close(); err != nil {
			log.Println(err)
		}
		ipcc.conn = nil	
	}
}

/*
//Todo : SendRecvとの統合が可能なので統合
func (ipcc *IpcClientController) SendRecvAsync(msg []byte) int {
	//Todo : ２つのAsyncが来た時に問題あり
	if ipcc.conn != nil {
		ipcc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

		_, err := ipcc.conn.Write(msg)
		if err != nil {
			log.Println(err)
			return 0
		}
	}

	go func() {
		_buf := make([]byte, consts.BUFF_MAX)

		ipcc.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := ipcc.conn.Read(_buf[:])
		if err != nil {
			log.Println(err)
			ipcc.ipcrecv_ch <- ""
		}
		select {
			case ipcc.ipcrecv_ch <- string(_buf[0:n]):
		}
	}()

	return 0
}

//
func (ipcc *IpcClientController) SendRecv(msg []byte) string {
	if ipcc.conn != nil {
		ipcc.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err := ipcc.conn.Write(msg)
		if err != nil {
			log.Println(err)
			return ""
		}
		//
		_buf := make([]byte, consts.BUFF_MAX)

		ipcc.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := ipcc.conn.Read(_buf[:])
		if err != nil {
			log.Println(err)
			return ""
		}
		//		ipcc.ipcrecv_ch <- string(_buf[0:n])
		return string(_buf[0:n])
	}
	log.Println("cannnot send : not conneccted")
	return ""
}
*/
//
func (ipcc *IpcClientController) SendRecvAsync2(msg []byte) int {
	select {
		case ipcc.send_ch <- msg: 
	}
	return 0
}
//
func (ipcc *IpcClientController) Run2() {

	ipcc.send_ch = make(chan []byte , 12)
	ipcc.read_ch = make(chan string, 12)
	ipcc.write_ch = make(chan []byte, 12)

	// Start reader.
	go ipcc.reader(ipcc.conn, ipcc.read_ch)

	// Start writer
	go ipcc.writer(ipcc.conn, ipcc.write_ch)
// Todo : 同期送信をどうする?
	//
	go func() {
		for {
			select {
				case _r  := <- ipcc.read_ch:
					ipcc.ipcrecv_ch <- _r
				case _w  := <- ipcc.send_ch:
					ipcc.write_ch <- _w
			}
		}
	}()
	
}

/*
//
func (ipcc *IpcClientController) Run(sch chan string, rch chan string) {
	go ipcc.ipcClientStart(sch)
}
*/

//
func (ipcc *IpcClientController) GetReceiveChannel() chan interface{} {
	return ipcc.ipcrecv_ch
}

//
func New(sf string) *IpcClientController {
	return &IpcClientController{
		sockFiles:  sf,
		ipcrecv_ch: make(chan interface{}, 12),
	}
}
