package main

//
import (
	consts "../consts"
	CommonError "../errs"
	mes "../message"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

//
func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.LockOSThread()
}

//
type IpcRecvCallback func(rm *Rmanager, msg string)

//
type IpcClient interface {
	Init() int
	connect() net.Conn
	Run() int
	recv_callback(msg string)
	Terminate() int
}

//
type IpcMessageHandler struct {
	Types   int
	Handler func([]byte, mes.MessageCommon)
}

//
type Rmanager struct {
	signal_ch chan os.Signal
	exit_ch   chan int
	recv_ch   chan string
	send_ch   chan []byte

	ipc_recv_callback IpcRecvCallback
	IpcMessageHandler []*IpcMessageHandler
}

//
func (rm *Rmanager) Init(ipcfn IpcRecvCallback, handler []*IpcMessageHandler) (ret int) {
	rm.signal_ch = make(chan os.Signal, 1)
	signal.Notify(rm.signal_ch, syscall.SIGTERM, syscall.SIGCHLD)

	rm.exit_ch = make(chan int)
	rm.recv_ch = make(chan string)
	rm.send_ch = make(chan []byte)

	rm.ipc_recv_callback = ipcfn
	rm.IpcMessageHandler = handler
	return 0
}

//
func (rm *Rmanager) recv_callback(msg string) {
	rm.ipc_recv_callback(rm, msg)
}

//
func (rm *Rmanager) Terminate() (ret int) {
	close(rm.exit_ch)
	close(rm.recv_ch)
	close(rm.send_ch)
	return 0
}

//
func (rm *Rmanager) connect() net.Conn {
	_c, err := net.Dial("unix", "/tmp/controller.sock")
	if err != nil {
		log.Println("Cannot connect IPC...")

		for i := 0; i < 3; i++ {
			log.Println("Reconnect.....")
			time.Sleep(3 * time.Second)
			_c, err = net.Dial("unix", "/tmp/echo.sock")
			if err == nil {
				break
			}
		}
	}
	CommonError.CheckErrorPanic(err, "Dial Error")
	return _c
}

//
func (rm *Rmanager) Run() (ret int) {
	_c := rm.connect()

	go rm.reader(_c)
	go rm.writer(_c)
	//Make Hello Request.
	_request := mes.MessageHello{
		Header: mes.MessageHeader{
			Destination_id: int(consts.CONTROLLER_ID),
			Source_id:      int(consts.RMANAGER_ID),
			Types:          int(mes.MESSAGE_ID_HELLO),
		},
		Pid:     os.Getpid(),
		Message: "HELLO",
	}
	//Send Hello Request.
	rm.SendMesage(mes.MakeMessage(_request))
	fmt.Println(_request)
	//
	go func() {
		for {
			select {
			case _s := <-rm.signal_ch:
				fmt.Println("SIGNALED")
				switch _s {
				case syscall.SIGTERM:
					rm.exit_ch <- 1
				case syscall.SIGCHLD:
					fmt.Println("CHILD EXIT")
				default:
					rm.exit_ch <- 1
				}
			case _s := <-rm.recv_ch:
				rm.recv_callback(_s)
			default:
			}
		}
	}()
	_ = <-rm.exit_ch

	return 0
}

//
func (rm *Rmanager) reader(r io.Reader) {
	for {
		_buf := make([]byte, consts.BUFF_MAX)
		_nr, err := r.Read(_buf[:])
		CommonError.CheckError(err, "Read ERROR")
		if err != nil {
			return
		}
		rm.recv_ch <- string(_buf[0:_nr])
	}
}

//
func (rm *Rmanager) writer(w io.Writer) {
	for {
		var _msg []byte
		select {
		case _msg = <-rm.send_ch:
		default:
			//                        msg = "hi"
		}
		if len(_msg) != 0 {
			fmt.Println("SEND : " + string(_msg))
			_, err := w.Write([]byte(_msg))
			CommonError.CheckError(err, "Write ERROR")
			if err != nil {
				break
			}
		}
		time.Sleep(1e9)
	}
}

//
func (rm *Rmanager) SendMesage(mes []byte) {
	rm.send_ch <- mes
}

//
func message_hello_handler(recv_mes []byte, head mes.MessageCommon) {
	var ms mes.MessageHello
	//Recv HelloMessage Unmarshal
	if err := json.Unmarshal(recv_mes, &ms); err != nil {
		fmt.Println("Unmarshal ERROR" + err.Error())
		return
	}
	fmt.Printf("RECV:")
	fmt.Println(ms)
}

//
func message_resource_handler(recv_mes []byte, head mes.MessageCommon) {
	var ms mes.MessageResourceControll
	//Recv MessageResourceControll Unmarshal
	if err := json.Unmarshal(recv_mes, &ms); err != nil {
		fmt.Println("Unmarshal ERROR" + err.Error())
		return
	}
	fmt.Printf("RECV:")
	fmt.Println(ms)
}

//
func ipc_recv_process(rm *Rmanager, msg string) {
	var _head mes.MessageCommon
	_recv_mes := []byte(msg)
	if err := json.Unmarshal(_recv_mes, &_head); err != nil {
		fmt.Println("unmarshal ERROR" + err.Error())
	}
	//
	var _processed bool = false
	fmt.Println("receive------")
	for i := 0; i < len(rm.IpcMessageHandler); i++ {
		//
		if rm.IpcMessageHandler[i].Types == _head.Header.Types {
			//
			rm.IpcMessageHandler[_head.Header.Types].Handler(_recv_mes, _head)
			_processed = true
			break
		}
	}
	if _processed == false {
		fmt.Println("receive Unkown MessageTypes")
	}
}

//
func NewRmanager(ipcfn IpcRecvCallback, handler []*IpcMessageHandler) *Rmanager {
	_rm := new(Rmanager)
	_rm.Init(ipcfn, handler)

	return (_rm)
}

//
func main() {

	_rm := NewRmanager(
		ipc_recv_process,
		//
		[]*IpcMessageHandler{
			{Types: mes.MESSAGE_ID_HELLO, Handler: message_hello_handler},
			{Types: mes.MESSAGE_ID_RESOUCE, Handler: message_resource_handler},
		},
	)

	_rm.Run()

	_rm.Terminate()
}
