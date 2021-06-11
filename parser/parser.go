package parser

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type server struct {
	ActiveDC string
	Version  int
	Ports    map[string]string
	Servers  map[string]ServerInfo
}
type ServerInfo struct {
	IP      string
	LocalIP string
	Alias   []string
}

func GetServerList() (server, error) {
	client := http.Client{}
	req, errR := http.NewRequest("GET", "http://hub.test.com:8080/config", nil)
	req.Header.Set("Content-Type", "applicaiton/json")
	resp, errR := client.Do(req)
	if errR != nil {
		return server{}, errR
	}

	body, errI := ioutil.ReadAll(resp.Body)
	if errI != nil {
		return server{}, errI
	}

	var s server
	errJ := json.Unmarshal(body, &s)
	if errJ != nil {
		return server{}, errJ
	}

	return s, nil

}
