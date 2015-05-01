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
func (bs *BaseControll) TerminateBase() {
	close(bs.Exit_ch)
	close(bs.Signal_ch)
}
