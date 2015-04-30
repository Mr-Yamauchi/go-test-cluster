package message
//
import (
	"testing"
	T1 "../message"
)
//
func TestMessage(t *testing.T){
        _struct := T1.MessageHello {
                        Header : T1.MessageHeader {
                                Destination_id : 10,
                                Source_id : 20,
                                Types : 1,
                        },
                        Message : "HELLO",
 
                 }
        _struct2 := T1.MessageHeader {
                                Destination_id : 10,
                                Source_id : 20,
                                Types : 1,                 }

	if ms := T1.MakeMessage(_struct); ms == nil {
		t.Errorf("json convert error")
	}
	if ms2 := T1.MakeMessage(_struct2); ms2 == nil {
		t.Errorf("json convert error")
	}
}
