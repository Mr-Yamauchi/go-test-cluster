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

var ChannelList []*ChannelHandlerData
//
func SetChannelHandler(list []*ChannelHandlerData, ct base.LockUnlocker,  data *ChannelHandlerData )([]*ChannelHandlerData) {
        ct.Lock()
        defer ct.Unlock()
        if list == nil {
                list = make([]*ChannelHandlerData, 0)
        }
        _channelList := append(list, data)

        return _channelList
}

//
func (cc ChannelHandlerData) Exec(cit interface{}, data interface{}) {
	cc.handler(cit, data)
}

//
func New(ch chan interface{}, fn func(cit interface{}, data interface{}))(*ChannelHandlerData){
	return & ChannelHandlerData { Ch : ch , handler : fn } 
}
