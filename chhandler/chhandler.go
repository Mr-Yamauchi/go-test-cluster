package chhandler

//
import (
	"../base"
)

//
type ChannelHandlerData struct {
        Ch      chan interface{}
        handler func(cit interface{}, data interface{})
}

type ChannelLists []*ChannelHandlerData

var ChannelList ChannelLists

type ChannelHandler interface {
	GetLen()(int)
	Exec(idx int, cit interface{}, data interface{}) 
	GetCh(idx int)(chan interface{})
}
//
func SetChannelHandler(list ChannelLists, ct base.LockUnlocker,  data *ChannelHandlerData )(ChannelLists) {
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
	cc[idx].handler(cit,data)
	
}

//
func (cc ChannelLists) GetCh(idx int)(chan interface{}) {
	return cc[idx].Ch
}

//
func (cc ChannelLists) GetLen()(int) {
	return len(cc)
}

//
func New(ch chan interface{}, fn func(cit interface{}, data interface{}))(*ChannelHandlerData){
	return & ChannelHandlerData { Ch : ch , handler : fn } 
}

//
