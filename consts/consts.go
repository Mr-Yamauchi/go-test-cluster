package consts

import 	"log/syslog"

const Logtag = "test"
const Logpriority = syslog.LOG_INFO | syslog.LOG_LOCAL1

//ProcessID
type ProcessId int
const (
	_ 	ProcessId = iota
	CONTROLLER_ID 
	RMANAGER_ID
)

//
const BUFF_MAX = 1024

//
func (s ProcessId) String() string {
	switch s {
	case CONTROLLER_ID:
		return "Controller"
	case RMANAGER_ID:
		return "Rmanager"
	default:
		return "Unknown"
	}
}
//StatusID
type StatusId int
const (
	_	 = iota
	STARTUP	
	ALL_CLIENT_UP
	LOAD_RESOURCE_SETTING
	CONTROL_RESOURCE
	PENDING
	OPERATIONAL
)
