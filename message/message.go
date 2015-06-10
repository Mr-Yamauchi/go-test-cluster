package message

//
import (
	"encoding/json"
	"fmt"
)

//
type MessageId int

const (
	MESSAGE_ID_HELLO = iota
	MESSAGE_ID_RESOUCE
	MESSAGE_ID_RESOUCE_RESPONSE
)

//
type MessageHeader struct {
	SeqNo	       uint64 `json:"seqno"`
	Destination_id int `json:"destination_id"`
	Source_id      int `json:"source_id"`
	Types          int `json:"types"`
}
type MessageCommon struct {
	Header MessageHeader `json:"header"`
	Body   interface{}       `json:"body"`
}
type MessageHello struct {
	Header  MessageHeader `json:"header"`
	MessageHelloBody      `json:"body"`
}
type MessageHelloBody struct {
	Pid     int           `json:"pid"`
	Message string        `json:"messge"`
}

type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type MessageResourceControllRequestBody struct {
	Rscid	      int	    `json:"rscid"`
	Operation     string        `json:"operation"`
	Resource_Name string        `json:"resource_name"`
	Interval      int64	    `json:"interval"`
	Timeout       int64         `json:"timeout"`	
	Delay         int64         `json:"delay"`
	Async         bool          `json:"async"`
	ParamLen      int	    `json:"paramlen"`
	Parameters    []Parameter   `json:"parameters"`
}
type MessageResourceControllRequest struct {
	Header        MessageHeader `json:"header"`
	MessageResourceControllRequestBody      `json:"body"`
}

type MessageResourceControllResponseBody struct {
	Pid           int           `json:"pid"`
	Rscid	      int	    `json:"rscid"`
	Operation     string        `json:"operation"`
	Resource_Name string        `json:"resource_name"`
	ResultCode   int	    `json:"resultcode"`
	Message string        `json:"messge"`
}
type MessageResourceControllResponse struct {
	Header        MessageHeader `json:"header"`
	MessageResourceControllResponseBody      `json:"body"`
}

//
func MakeMessage(data interface{}) []byte {
	_json, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return _json
}

//
func ParametersToString( len int, p []Parameter )[]string {
	r := []string {}
	for i := 0; i < len; i++ {
		r = append(r, p[i].Name + "=" + p[i].Value)
fmt.Println("r = ", r)
	}
	return r
}

/*
func main(){
	_struct := MessageHello {
			Header : MessageHeader {
				Destination_id : 10,
				Source_id : 20,
				Types : 1,
			},
			Message : "HELLO",

		 }
	_struct2 := MessageHeader {
				Destination_id : 10,
				Source_id : 20,
				Types : 1,
		 }

	fmt.Println(string(MakeMessage(_struct)))
	fmt.Println(string(MakeMessage(_struct2)))


	var t MessageHello
	if err := json.Unmarshal(MakeMessage(_struct), &t); err != nil {
		fmt.Println("EERRO"+err.Error())
	}
	fmt.Println(t.Message)


}
*/
