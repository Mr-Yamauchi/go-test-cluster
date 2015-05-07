package chhandler

//
import (
	"../base"
	"../consts"
	"fmt"
	"syscall"
)

//
type ChannelHandlerData struct {
	Ch      chan interface{}
	handler func(cit interface{}, data interface{})
}

type ChannelLists []*ChannelHandlerData

var ChannelList ChannelLists

type ChannelHandler interface {
	GetLen() int
	Exec(idx int, cit interface{}, data interface{})
	GetCh(idx int) chan interface{}
}

//
func SetChannelHandler(list ChannelLists, ct base.LockUnlocker, data *ChannelHandlerData) ChannelLists {
	ct.Lock()
	defer ct.Unlock()
	if list == nil {
		list = make(ChannelLists, 0)
	}
	_channelList := append(list, data)

	return _channelList
}

//
func (cc ChannelLists) Exec(idx int, cit interface{}, data interface{}) {
	cc[idx].handler(cit, data)

}

//
func (cc ChannelLists) GetCh(idx int) chan interface{} {
	return cc[idx].Ch
}

//
func (cc ChannelLists) GetLen() int {
	return len(cc)
}

//
func New(ch chan interface{}, fn func(cit interface{}, data interface{})) *ChannelHandlerData {
	return &ChannelHandlerData{Ch: ch, handler: fn}
}

//
func ProcessRun(ct base.Runner, chData ChannelHandler) (wait int) {
	//
	go func() {
		for {
			ct.Lock()
			for idx := 0; idx < chData.GetLen(); idx++ {
				select {
				case _sig_ch := <-ct.GetSignalChannel():
					fmt.Println("SIGNALED")
					switch _sig_ch {
					case syscall.SIGTERM:
						ct.GetExitChannel() <- 1
					case syscall.SIGCHLD:
						fmt.Println("CHILD EXIT")
					default:
						ct.GetExitChannel() <- 1
					}
				case _ch := <-chData.GetCh(idx):
					chData.Exec(idx, ct, _ch)
				default:
				}
			}
			ct.Unlock()
		}
	}()
	if _ch := ct.GetStatusChannel(); _ch != nil {
		_ch <- consts.STARTUP
	}
	wait = <-ct.GetExitChannel()

	return
}
