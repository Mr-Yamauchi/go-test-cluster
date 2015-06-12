package corosync

/*
#cgo LDFLAGS: -lcpg -lquorum

#include <inttypes.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/select.h>
#include <sys/uio.h>
#include <sys/un.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <time.h>
#include <sys/time.h>
#include <assert.h>
#include <limits.h>

#include <corosync/corotypes.h>
#include <corosync/cpg.h>
#include <corosync/quorum.h>

#define retrybackoff(counter) {    \
		counter++;                    \
		printf("Restart operation after %ds\n", counter); \
		sleep((unsigned int)counter);               \
		restart = 1;			\
		continue;			\
}

#define cs_repeat_init(counter, max, code) do {    \
	code;                                 \
	if (result == CS_ERR_TRY_AGAIN || result == CS_ERR_QUEUE_FULL || result == CS_ERR_LIBRARY) {  \
		counter++;                    \
		printf("Retrying operation after %ds\n", counter); \
		sleep((unsigned int)counter);               \
	} else {                              \
		break;                        \
	}                                     \
} while (counter < max)

#define cs_repeat(counter, max, code) do {    \
	code;                                 \
	if (result == CS_ERR_TRY_AGAIN || result == CS_ERR_QUEUE_FULL) {  \
		counter++;                    \
		printf("Retrying operation after %ds\n", counter); \
		sleep((unsigned int)counter);               \
	} else {                              \
		break;                        \
	}                                     \
} while (counter < max)

static cpg_handle_t handle;
static quorum_handle_t q_handle;
static int quit = 0;
static int show_ip = 0;
static int restart = 0;
static uint32_t nodeidStart = 0;
static int have_quorum = 0;

static uint32_t get_cpg_local_id();
static void sendClusterMessage(char *msg);
static int runs();

void corosyncDeliverCallback(uint32_t nodeid, uint32_t pid, size_t msg_len, char *msg);

static uint32_t get_cpg_local_id()
{
	uint32_t nodeid,
	result = cpg_local_get(handle, &nodeid);
	return nodeid;

}
static void DeliverCallback (
	cpg_handle_t handle,
	const struct cpg_name *groupName,
	uint32_t nodeid,
	uint32_t pid,
	void *msg,
	size_t msg_len)
{
	//call go function
	corosyncDeliverCallback(nodeid, pid, msg_len, (char *)msg);
}

static struct cpg_address *getMember(struct cpg_address *ptr, uint32_t idx);
static struct cpg_address *getMember(struct cpg_address *ptr, uint32_t idx) {
	return (ptr+idx);
}

void corosyncConfchgCallback(
	struct cpg_address *member_list,
	size_t member_list_entries,
	struct cpg_address *left_list,
	size_t left_list_entries,
	struct cpg_address *joined_list,
	size_t joined_list_entries);

static void ConfchgCallback (
	cpg_handle_t handle,
	const struct cpg_name *groupName,
	const struct cpg_address *member_list, size_t member_list_entries,
	const struct cpg_address *left_list, size_t left_list_entries,
	const struct cpg_address *joined_list, size_t joined_list_entries)
{
	unsigned int i;
	int result;
	uint32_t nodeid;

	result = cpg_local_get(handle, &nodeid);
	if(result != CS_OK) {
		printf("failed to get local nodeid %d\n", result);
		nodeid = 0;
	}
	if (left_list_entries && (pid_t)left_list[0].pid == getpid()) {
		printf("We might have left the building pid %d\n", left_list[0].pid);
		if(nodeidStart) {
			if(htonl((uint32_t)nodeid) == INADDR_LOOPBACK) {
				printf("We probably left the building switched identity? start nodeid %d nodeid %d current nodeid %d pid %d\n", nodeidStart, left_list[0].nodeid, nodeid, left_list[0].pid);
			} else if(htonl((uint32_t)left_list[0].nodeid) == INADDR_LOOPBACK) {
				printf("We probably left the building started alone? start nodeid %d nodeid %d current nodeid %d pid %d\n", nodeidStart, left_list[0].nodeid, nodeid, left_list[0].pid);
			}
			if(left_list[0].nodeid == nodeidStart) {
				printf("We have left the building direct match start nodeid %d nodeid %d local get current nodeid %d pid %d\n", nodeidStart, left_list[0].nodeid, nodeid, left_list[0].pid);
				// quit = 1;
				restart = 1;
			} else {
				printf("Probably another node with matching pid start nodeid %d nodeid %d current nodeid %d pid %d\n", nodeidStart, left_list[0].nodeid, nodeid, left_list[0].pid);
			}
		}
	}
	//call go function
	corosyncConfchgCallback(
		(struct cpg_address *)member_list, member_list_entries,
		(struct cpg_address *)left_list, left_list_entries,
		(struct cpg_address *)joined_list, joined_list_entries);
}

static uint32_t *getMember2(uint32_t *ptr, uint32_t idx);
static uint32_t *getMember2(uint32_t *ptr, uint32_t idx) {
	return (ptr+idx);
}

void corosyncTotemchgCallback(
        struct cpg_ring_id ring_id,
        uint32_t member_list_entries,
        uint32_t *member_list);

static void TotemConfchgCallback (
	cpg_handle_t handle,
        struct cpg_ring_id ring_id,
        uint32_t member_list_entries,
        const uint32_t *member_list)
{
	//call go function
	corosyncTotemchgCallback(ring_id, member_list_entries, (uint32_t *)member_list);

}

static cpg_model_v1_data_t model_data = {
	.cpg_deliver_fn =            DeliverCallback,
	.cpg_confchg_fn =            ConfchgCallback,
	.cpg_totem_confchg_fn =      TotemConfchgCallback,
	.flags =                     CPG_MODEL_V1_DELIVER_INITIAL_TOTEM_CONF,
};

void corosyncQuorumchgCallback(uint32_t quorate);
static void
quorum_notification(quorum_handle_t handle,
        uint32_t quorate,
        uint64_t ring_id, uint32_t view_list_entries, uint32_t * view_list) {

	//
	corosyncQuorumchgCallback(quorate);
	
}
static quorum_callbacks_t quorum_callback = {
	.quorum_notify_fn = quorum_notification
};

static struct cpg_name group_name;

//
void sendClusterMessage(char *msg){ 
	char inbuf[132];
	struct iovec iov;

	iov.iov_base = msg;
	iov.iov_len = strlen(msg)+1;
	cpg_mcast_joined(handle, CPG_TYPE_AGREED, &iov, 1);
}
//
static int connect_quorum( quorum_handle_t *h ) {
	int rc;
	uint32_t type = 0;
	int quorate = 0;

	
	if ( CS_OK != (rc = quorum_initialize(h, &quorum_callback, &type))) goto exit_label;

	if ( type != QUORUM_SET ) goto exit_label;
	
	if ( CS_OK != (rc = quorum_getquorate(*h, &quorate))) goto exit_label;

    	if ( CS_OK != (rc = quorum_trackstart(*h, CS_TRACK_CHANGES | CS_TRACK_CURRENT))) goto exit_label;
 
exit_label:
	if ( rc != CS_OK) {
		printf("ERROR");
	}

	return(rc);
}

int runs () {
	fd_set read_fds;
	int max_fd = 0;
	int select_fd;
	int select_fd_quorum;
	int result;
	int retries;
	const char *options = "i";
	int opt;
	unsigned int nodeid;
	char *fgets_res;
	struct cpg_address member_list[64];
	int member_list_entries;
	int i;
	int recnt;
	int doexit;

	doexit = 0;

	strcpy(group_name.value, "GROUP");
	group_name.length = 6;

	recnt = 0;

	restart = 1;

	do {
		if(restart) {
			restart = 0;
			retries = 0;
			cs_repeat_init(retries, 30, result = cpg_model_initialize (&handle, CPG_MODEL_V1, (cpg_model_data_t *)&model_data, NULL));
			if (result != CS_OK) {
				printf ("Could not initialize Cluster Process Group API instance error %d\n", result);
				retrybackoff(recnt);
			}
			retries = 0;
			cs_repeat(retries, 30, result = cpg_local_get(handle, &nodeid));
			if (result != CS_OK) {
				printf ("Could not get local node id\n");
				retrybackoff(recnt);
			}
			nodeidStart = nodeid;

			retries = 0;
			cs_repeat(retries, 30, result = cpg_join(handle, &group_name));
			if (result != CS_OK) {
				printf ("Could not join process group, error %d\n", result);
				retrybackoff(recnt);
			}

			retries = 0;
			cs_repeat(retries, 30, result = cpg_membership_get (handle, &group_name,
				(struct cpg_address *)&member_list, &member_list_entries));
			if (result != CS_OK) {
				printf ("Could not get current membership list %d\n", result);
				retrybackoff(recnt);
			}
			recnt = 0;

#if 0
			printf ("membership list\n");
			for (i = 0; i < member_list_entries; i++) {
				printf ("node id %d pid %d\n", member_list[i].nodeid,
					member_list[i].pid);
			}
#endif
			FD_ZERO (&read_fds);
			cpg_fd_get(handle, &select_fd);

			result =  connect_quorum(&q_handle);
			if (result != CS_OK) {
				perror ("connect_quorum\n");
			}
			quorum_fd_get(q_handle, &select_fd_quorum);

		}
		FD_SET (select_fd, &read_fds);
		FD_SET (select_fd_quorum, &read_fds);
		if (select_fd > select_fd_quorum) {  
			max_fd = select_fd;
		} else {
			max_fd = select_fd_quorum;
		}

		result = select (max_fd + 1, &read_fds, 0, 0, 0);
		if (result == -1) {
			perror ("select\n");
		}

		if (FD_ISSET (select_fd, &read_fds)) {
			if (cpg_dispatch (handle, CS_DISPATCH_ALL) != CS_OK) {
				if(doexit) {
					exit(1);
				}
				restart = 1;
			}
		}
		if (FD_ISSET (select_fd_quorum, &read_fds)) {
			if (quorum_dispatch (q_handle, CS_DISPATCH_ALL) != CS_OK) {
				if(doexit) {
					exit(1);
				}
				restart = 1;
			}
		}
		if(restart) {
			if(!doexit) {
				result = cpg_finalize (handle);
				printf ("Finalize+restart result is %d (should be 1)\n", result);
				continue;
			}
		}
	} while (result && !quit && !doexit);

	result = cpg_finalize (handle);
	printf ("Finalize  result is %d (should be 1)\n", result);
	return (0);
}
*/
import "C"
import "strings"
import "../debug"

//
var coro_cfg_chan chan interface{}
var coro_deliv_chan chan interface{}
var coro_totem_chan chan interface{}
var coro_quorum_chan chan interface{}

//
type ListData struct {
	Nodeid	uint
	Pid	uint
}
//
type CorosyncConfchg struct {
	Member_list []ListData
	Left_list []ListData
	Join_list []ListData
}

//Need export for C-call.
//export corosyncConfchgCallback
func corosyncConfchgCallback(member_list *C.struct_cpg_address, member_list_entries C.size_t,
        left_list *C.struct_cpg_address, left_list_entries C.size_t,
        join_list *C.struct_cpg_address, joined_list_entries C.size_t) {

	_m := CorosyncConfchg{}

	for i:= 0; i<int(member_list_entries);i++ {
		p := (*C.struct_cpg_address)(C.getMember(member_list, C.uint32_t(i)))
		_m.Member_list = append(_m.Member_list, ListData{ uint(p.nodeid), uint(p.pid) })	
	}
	for j:= 0; j<int(left_list_entries);j++ {
		p := (*C.struct_cpg_address)(C.getMember(left_list, C.uint32_t(j)))
		_m.Left_list = append(_m.Left_list, ListData{ uint(p.nodeid), uint(p.pid) })	
	}
	for k:= 0; k<int(joined_list_entries);k++ {
		p := (*C.struct_cpg_address)(C.getMember(join_list, C.uint32_t(k)))
		_m.Join_list = append(_m.Join_list, ListData{ uint(p.nodeid), uint(p.pid) })	
	}

	coro_cfg_chan <- _m
}

//
type CorosyncDeliver struct {
	Nodeid 	uint
	Pid	uint
	Msg	string
}

//Need export for C-call.
//export corosyncDeliverCallback
func corosyncDeliverCallback(nodeid C.uint32_t, pid C.uint32_t, msg_len C.size_t, msg *C.char) {
	debug.DEBUGT.Println("------------golang:Deliver------------")
	debug.DEBUGT.Println("DeliverCallback: message (len=%d)from %s: %d:%d\n",
		       msg_len, strings.Trim(C.GoString(msg), "\n"), nodeid, pid)

	coro_deliv_chan <- CorosyncDeliver { 
		Nodeid : uint(nodeid), 
		Pid : uint(pid), 
		Msg : C.GoString(msg),
	}
}

//
type CorosyncTotemchg struct {
	Member_list []uint
}

//Need export for C-call.
//export corosyncTotemchgCallback
func corosyncTotemchgCallback(ring_id C.struct_cpg_ring_id,
        member_list_entries C.uint32_t,
        member_list *C.uint32_t) {

	_m := CorosyncTotemchg{}

	debug.DEBUGT.Println("------------golang:TotemchgCallback------------")
/*
	debug.DEBUGT.Printf("goTotemConfchgCallback: ringid (%d.%d)\n", ring_id.nodeid, ring_id.seq)
	debug.DEBUGT.Printf("active processors %d: \n", member_list_entries)
*/

	for i:=0; i<int(member_list_entries); i++ {
		p := (*C.uint32_t)(C.getMember2(member_list, C.uint32_t(i)))
		_m.Member_list = append(_m.Member_list, uint(*p))
	}
	coro_totem_chan <- _m
}

//Need export for C-call.
//export corosyncQuorumchgCallback
func corosyncQuorumchgCallback(quorate C.uint32_t) {
	coro_quorum_chan <- uint(quorate)
}

//
func SendClusterMessage(msg string) {
	C.sendClusterMessage(C.CString(msg))
}

//
func GetLocalId()(uint) {
	return uint(C.get_cpg_local_id())
}
//
func Init()(chan interface{}, chan interface{}, chan interface{}, chan interface{}) {

	coro_cfg_chan = make(chan interface{},128)
	coro_deliv_chan = make(chan interface{},128)
	coro_totem_chan = make(chan interface{},128)
	coro_quorum_chan = make(chan interface{},128)

	return coro_cfg_chan, coro_deliv_chan, coro_totem_chan, coro_quorum_chan
}

//
func Run() {
	go func() {
		C.runs()
	}()
}
