package confgure

//
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"strconv"
)

//
type ControlConfig struct {
	Port         int    `json:"port"`
	Cluster_name string `json:"cluster_name"`
	Max_buffer   int    `json:max_buffer`
}

//
func New(filename string) (ret *ControlConfig) {
	var _config ControlConfig
	_file, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Read Error")
		return nil
	}

	json.Unmarshal(_file, &_config)
	return &_config
}

//
func (cfg ControlConfig) DumpConfig() {
	//fmt.Println(" port := ", strconv.Itoa(cfg.Port))
	//fmt.Println(" cluster_name := ", cfg.Cluster_name)
	fmt.Println(cfg)
}
