package chhandler

//
import (
	"../base"
)

//
type ChannelHandlerData struct {
        Ch      chan interface{}
        Handler func(cit interface{}, data interface{})
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
