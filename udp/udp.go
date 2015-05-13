package udp

//
import (
	consts "../consts"
	"log"
	"net"
	"time"
)

type UdpController interface {
	GetUdpChannel() (chan string, chan interface{})
	Run(sc chan string, rc chan interface{})
}

type senderStarter interface {
	senderStart(ch chan string)
}

type receiverStarter interface {
	receiverStart(ch chan interface{})
}


type UdpControll struct {
	udpsend_ch    chan string
	udprecv_ch    chan interface{}
}

//
func (udc *UdpControll) SenderStart(exec senderStarter) {
	exec.senderStart(udc.udpsend_ch)
}

//
func (udc *UdpControll) ReceiverStart(exec receiverStarter) {
	exec.receiverStart(udc.udprecv_ch)
}

//
type executer func()

//
func (exe executer) senderStart(ch chan string) {

	_ServerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	if err != nil {
		log.Println(err.Error())
	}

	_LocalAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		log.Println(err.Error())
	}

	_Conn, err := net.DialUDP("udp", _LocalAddr, _ServerAddr)
	if err != nil {
		log.Println(err.Error())
	}

	defer func() {
		if err := _Conn.Close(); err != nil {
			log.Println(err.Error())
		}
	}()
	i := 0
	for {
		var _msg string = ""
		select {
		case _msg = <-ch:
		default:
			_msg = "AAA"
		}
		i++
		_buf := []byte(_msg)
		_, err := _Conn.Write(_buf)
		if err != nil {
			log.Println(_msg, err.Error())
		}
		time.Sleep(time.Second * 1)
	}
}

//
func (exe executer) receiverStart(ch chan interface{}) {
	/* Lets prepare a address at any address at port 10001*/
	_ServerAddr, err := net.ResolveUDPAddr("udp", ":10001")
	if err != nil {
		log.Println(err.Error())
	}

	/* Now listen at selected port */
	_ServerConn, err := net.ListenUDP("udp", _ServerAddr)
	if err != nil {
		log.Println(err.Error())
	}
	defer func() {
		if err := _ServerConn.Close(); err != nil {
			log.Println("close ServerConn error", err.Error())
		}
	}()

	_buf := make([]byte, consts.BUFF_MAX)

	for {
		_nr, _addr, err := _ServerConn.ReadFromUDP(_buf)
		_msg := string(_buf[0:_nr]) + _addr.String()
		ch <- _msg

		if err != nil {
			log.Println(err.Error())
		}
	}
}

//
func (udc *UdpControll) GetUdpChannel() (chan string, chan interface{}) {
	return udc.udpsend_ch, udc.udprecv_ch
}

//
func New() *UdpControll {
	return &UdpControll{
		udpsend_ch:    make(chan string),
		udprecv_ch:    make(chan interface{}),
	}
}

//
func (udc *UdpControll) Run(sc chan string, rc chan interface{}) {
	var e executer
	//
	go udc.ReceiverStart(e)
	//
	go udc.SenderStart(e)
}
