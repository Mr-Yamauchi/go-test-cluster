package corosync

/*
#cgo LDFLAGS: -lcpg

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

#ifndef HOST_NAME_MAX
#define HOST_NAME_MAX _POSIX_HOST_NAME_MAX
#endif

static cpg_handle_t handle;
static int quit = 0;
static int show_ip = 0;
static int restart = 0;
static uint32_t nodeidStart = 0;

static void sendClusterMessage(char *msg);
static int runs();

void corosyncDeliverCallback(uint32_t nodeid, uint32_t pid, size_t msg_len, char *msg);

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

static struct cpg_name group_name;

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

void sendClusterMessage(char *msg){ 
	char inbuf[132];
	struct iovec iov;

	iov.iov_base = msg;
	iov.iov_len = strlen(msg)+1;
	cpg_mcast_joined(handle, CPG_TYPE_AGREED, &iov, 1);
}

int runs () {
	fd_set read_fds;
	int select_fd;
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
			printf ("Local node id is %x\n", nodeid);
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
		}
		FD_SET (select_fd, &read_fds);

		result = select (select_fd + 1, &read_fds, 0, 0, 0);
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

var t1 chan interface{}
var t2 chan interface{}
var t3 chan interface{}

//
type ListData struct {
	Nodeid	uint
	Pid	uint
}
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

	t1 <- _m
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

	t2 <- CorosyncDeliver { 
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
	t3 <- _m
}

//
func SendClusterMessage(msg string) {
	C.sendClusterMessage(C.CString(msg))
}

//
func Init()(chan interface{}, chan interface{}, chan interface{}) {
	t1 = make(chan interface{},128)
	t2 = make(chan interface{},128)
	t3 = make(chan interface{},128)
	return t1, t2, t3
}

//
func Run() {
	go func() {
		C.runs()
	}()
}
