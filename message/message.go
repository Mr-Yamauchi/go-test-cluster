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
	Destination_id int `json:"destination_id"`
	Source_id      int `json:"source_id"`
	Types          int `json:"types"`
}
type MessageCommon struct {
	Header MessageHeader `json:"header"`
	Body   string        `json:"body"`
}
type MessageHello struct {
	Header  MessageHeader `json:"header"`
	Pid     int           `json:"pid"`
	Message string        `json:"messge"`
}
type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type MessageResourceControllRequest struct {
	Header        MessageHeader `json:"header"`
	Operation     string        `json:"operation"`
	Resource_Name string        `json:"resource_name"`
	Parameters    []Parameter   `json:"parameters"`
}
type MessageResourceControllResponse struct {
	Header        MessageHeader `json:"header"`
	Pid     int           `json:"pid"`
	Message string        `json:"messge"`
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
