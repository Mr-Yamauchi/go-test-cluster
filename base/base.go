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
	GetSignalChannel() chan os.Signal
	GetExitChannel() chan int
}

//
type Runner interface {
	Lock()
	Unlock()
	GetSignalChannel() chan os.Signal
	GetExitChannel() chan int
	GetStatusChannel() chan interface{}
}

//
type BaseControll struct {
	ChMutex   *sync.Mutex
	Signal_ch chan os.Signal
	Exit_ch   chan int
	Status_ch chan interface{}
}

//
func (bse *BaseControll) InitBase(sigs ...os.Signal) {
	bse.ChMutex = new(sync.Mutex)
	bse.Signal_ch = make(chan os.Signal, 1)
	signal.Notify(bse.Signal_ch, sigs...)
	bse.Exit_ch = make(chan int)
	bse.Status_ch = make(chan interface{}, 2)
}

//
func (bse BaseControll) Lock() {
	bse.ChMutex.Lock()
}

//
func (bse BaseControll) Unlock() {
	bse.ChMutex.Unlock()
}

//
func (bse BaseControll) GetSignalChannel() chan os.Signal {
	return bse.Signal_ch
}

//
func (bse BaseControll) GetExitChannel() chan int {
	return bse.Exit_ch
}

//
func (bse BaseControll) GetStatusChannel() chan interface{} {
	return bse.Status_ch
}

//
func (bse *BaseControll) TerminateBase() {
	close(bse.Exit_ch)
	close(bse.Signal_ch)
	close(bse.Status_ch)
}
