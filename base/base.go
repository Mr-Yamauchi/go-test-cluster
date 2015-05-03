package base

//
import (
	"os"
	"os/signal"
	"sync"
)

//
type LockUnlocker interface {
        Lock()
        Unlock()
}
type BaseController interface {
	LockUnlocker
	GetSignalChannel()(chan os.Signal)
	GetExitChannel()(chan int)
}

//
type BaseControll struct {
	ChMutex	  *sync.Mutex
	Signal_ch chan os.Signal
	Exit_ch   chan int
}

//
func (bs *BaseControll) InitBase(sigs... os.Signal) {
	bs.ChMutex = new(sync.Mutex)
	bs.Signal_ch = make(chan os.Signal, 1)
	signal.Notify(bs.Signal_ch, sigs...)
	bs.Exit_ch = make(chan int)
}

//
func (bs BaseControll) Lock() {
	bs.ChMutex.Lock()
}

//
func (bs BaseControll) Unlock() {
	bs.ChMutex.Unlock()
}

//
func (bs BaseControll)GetSignalChannel()(chan os.Signal) {
	return bs.Signal_ch
}

//
func (bs BaseControll)GetExitChannel()(chan int) {
	return bs.Exit_ch
}
//
func (bs *BaseControll) TerminateBase() {
	close(bs.Exit_ch)
	close(bs.Signal_ch)
}

