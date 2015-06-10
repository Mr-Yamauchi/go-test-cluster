package chhandler

//
import (
	"../debug"
	"../base"
	"../consts"
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
func (chl ChannelLists) Exec(idx int, cit interface{}, data interface{}) {
	chl[idx].handler(cit, data)

}

//
func (chl ChannelLists) GetCh(idx int) chan interface{} {
	return chl[idx].Ch
}

//
func (chl ChannelLists) GetLen() int {
	return len(chl)
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
					debug.DEBUGT.Println("SIGNALED")
					switch _sig_ch {
					case syscall.SIGTERM:
						ct.GetExitChannel() <- 1
					case syscall.SIGCHLD:
						debug.DEBUGT.Println("CHILD EXIT")
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
