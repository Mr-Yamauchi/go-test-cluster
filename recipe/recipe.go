package recipe

//
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

//
type Parameter struct {
	Name string 		`json:"name"`
	Value string		`json:"value"`
}
//
type Operation struct {
	Opname string 		`json:"opname"`
	Timeout int		`json:"timeout"`
	Interval int		`json:"interval"`
	Onfail string 		`json:"onfail"`
	
}
//
type Resource struct {
	Rscid	string		`json:"rscid"`	
	Type	string 		`json:"type"`
	Provider string		`json:"provider"`
	Name	string		`json:"name"`
	Params  []Parameter	`json:"parameters"`
	Op	[]Operation	`json:"operations"`
}
//
type ClusterResources struct {
	Resources []Resource 	`json:"resources"`
}
//
var instance *ClusterResources 

//
func New(filename string) (ret *ClusterResources) {
	
	if (instance == nil ) {
		instance = &ClusterResources{}
	}	

	_file, _err := ioutil.ReadFile(filename)
	if _err != nil {
		fmt.Println("read Error", _err.Error())
		return nil
	}

	_err = json.Unmarshal(_file, instance)
	if _err != nil {
		fmt.Println("unmarshal Error", _err.Error())
		return nil
	}
	return instance
	
}

//
func (cres ClusterResources) DumpResource(){
	fmt.Println("### ClusterResources ####", cres)
}

//
/*
func main(){
	a := New("resource.json")
	a.DumpResource()
	//fmt.Println(len(a.Resources))
	//fmt.Println(len(a.Resources[0].Op))
}
*/
