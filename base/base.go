package base

//
import (
	"os"
	"os/signal"
)

//
type BaseControll struct {
	Signal_ch chan os.Signal
	Exit_ch   chan int
}

//
func (bs *BaseControll) InitBase(sigs... os.Signal) {
	bs.Signal_ch = make(chan os.Signal, 1)
	signal.Notify(bs.Signal_ch, sigs...)
	bs.Exit_ch = make(chan int)
}

//
func (bs *BaseControll) TerminateBase() {
	close(bs.Exit_ch)
	close(bs.Signal_ch)
}
