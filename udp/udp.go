package udp

//
import (
	consts "../consts"
	"net"
	"time"
	"log"
)

type UdpController interface {
	GetUdpChannel()(chan string, chan interface{})
	Run(sc chan string, rc chan interface{})
}
type UdpSender interface {
	senderStart(ch chan string)
}
type UdpReceiver interface {
	receiverStart(ch chan interface{})
}

type SenderStarter func()
type RecieverStarter func()

type UdpControll struct {
	udpsend_ch chan string
	udprecv_ch chan interface{}
	senderStart   func(ch chan string)
	receiverStart func(ch chan interface{})
}

//
func (uc *UdpControll) SenderStart() {
	uc.senderStart(uc.udpsend_ch)
}

//
func (uc *UdpControll) ReceiverStart() {
	uc.receiverStart(uc.udprecv_ch)
}

//
func senderStart(ch chan string) {

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
func receiverStart(ch chan interface{}) {
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
func (uc *UdpControll) GetUdpChannel()(chan string, chan interface{}) {
	return uc.udpsend_ch, uc.udprecv_ch
}
//
func New() *UdpControll {
	return & UdpControll {
		senderStart : senderStart,
		receiverStart : receiverStart,
       		udpsend_ch : make(chan string),
        	udprecv_ch : make(chan interface{}),
	}
}

//
func (uc *UdpControll) Run(sc chan string, rc chan interface{}) {
	go uc.ReceiverStart()
	go uc.SenderStart()
}
