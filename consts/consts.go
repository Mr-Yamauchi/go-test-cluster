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

// Returnc Code
const CL_NORMAL_END = 0
const CL_ERROR = -1 
const CL_NOT_FORK = -2
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

//
func (s StatusId) String() string {
	var _ss string
	switch s {
	case STARTUP :
		_ss = "STARTUP"
	case ALL_CLIENT_UP :
		_ss = "ALL_CLIENT_UP"
	case LOAD_RESOURCE_SETTING :
		_ss = "LOAD_RESOURCE_SETTING"
	case CONTROL_RESOURCE :
		_ss = "CONTROL_RESOURCE"
	case PENDING :
		_ss = "PENDING"
	case OPERATIONAL :
		_ss = "OPERAIONAL"
	default : 
		_ss = ""
	}
	return _ss
}
