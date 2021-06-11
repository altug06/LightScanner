package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type HostInfo struct {
	Ports  []string
	Domain []string
	IP     string
}

type MailBody struct {
	Body    string `json:"body"`
	Subject string `json:"subject"`
	To      string `json:"to"`
	Html    bool   `json:"html"`
}

func sendMail(body string) {
	client := &http.Client{}
	m := &MailBody{
		Body: body,
		To:   "altug@test.com",
		Html: true,
	}
	m.Subject = "Port scan results"

	b, errJ := json.Marshal(m)
	if errJ != nil {
		log.Fatal(errJ)
	}

	fmt.Println(string(b))
	apiUrl := "https://api.test.com"
	resource := "/enterprise/proxy/raw"
	data := url.Values{}
	data.Set("token", "KAhsHV9BQWM72DEjk3gdvM4BQvLMVRmcQ")
	data.Set("JSON", string(b))

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := u.String()

	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode()))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	fmt.Println(data.Encode())

	resp, errR := client.Do(r)
	if errR != nil {
		log.Fatal(errR)
	}
	defer resp.Body.Close()

	io.Copy(ioutil.Discard, resp.Body)
	fmt.Println(resp.StatusCode)

}

func main() {
	var hosts []HostInfo
	h1 := HostInfo{
		IP: "1.2.3.4",
	}
	//   h1.Ports = make([]string, 5)
	//   h1.Domain = make([]string, 2)
	h1.Ports = []string{"22", "50", "60", "80", "8080"}
	h1.Domain = []string{"test.com", "test2.com"}

	hosts = append(hosts, h1)

	//   h2 := HostInfo{
	// 	Ports: string{"992","60","70","80","443"},
	// 	Domain: string{"test2.com","test3.com", "test4.com.tr"},
	// 	IP: "1.2.3.5",
	//   }

	//   hosts = append(hosts, h2)
	t, err := template.ParseFiles("/root/port-scanner/scane_template.html")
	if err != nil {
		fmt.Println(err.Error())
	}

	buf := new(bytes.Buffer)
	t.Execute(buf, hosts)
	sendMail(buf.String())

}
