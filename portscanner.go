package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"./parser"
	"./worker"
)

var client *http.Client

type MailBody struct {
	Body    string `json:"body"`
	Subject string `json:"subject"`
	To      string `json:"to"`
	Html    bool   `json:"html"`
}

var ignoredServers = []string{"geu-lb", "go-lb"}

func sendMail(body string) {

	m := &MailBody{
		Body: body,
		To:   "security@test.com",
		Html: true,
	}
	m.Subject = "Port scan results : " + time.Now().Format("01-02-2006 15:04:05")
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

func checkDomain(domain string) bool {
	for _, d := range ignoredServers {
		if d == domain {
			return true
		}
	}
	return false
}

func main() {

	TLSConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		},
		NextProtos: []string{"https"},
	}

	client = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			MaxIdleConns:          30,
			MaxIdleConnsPerHost:   30,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			TLSClientConfig:       TLSConfig,
		},
	}

	var logFile string
	flag.StringVar(&logFile, "log-file", "", "Location of the file to output the result of the scan")
	flag.Parse()
	ipPool := make(chan worker.DomIpPair, 100)
	wg := sync.WaitGroup{}

	workerPool := worker.NewPool(30, ipPool)
	workerPool.InitializeWorkers(&wg, logFile)

	s, err := parser.GetServerList()
	if err != nil {
		fmt.Printf("someting went wrong: %s", err.Error())
	}

	var servers map[string][]string
	servers = make(map[string][]string)
	fmt.Println(len(s.Servers))
	for domain, server := range s.Servers {
		if !checkDomain(domain) {
			servers[server.IP] = append(servers[server.IP], domain)
		}
	}

	for ip, doms := range servers {
		wg.Add(1)
		ipPool <- worker.DomIpPair{
			IP:     ip,
			Domain: doms,
		}
	}

	wg.Wait()

	for ip, ports := range worker.Openports {
		if len(ports) == 0 {
			delete(worker.Openports, ip)
		}
	}

	t, err := template.ParseFiles("/root/port-scanner/scane_template.html")
	if err != nil {
		fmt.Println(err.Error())
	}

	buf := new(bytes.Buffer)
	t.Execute(buf, worker.Hosts)
	sendMail(buf.String())

	// if len(worker.Openports) != 0 {
	// 	b, err := json.Marshal(worker.Openports)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	errF := ioutil.WriteFile(logFile, b, 0644)
	// 	if errF != nil {
	// 		fmt.Println(fmt.Errorf("could not write to file, :%v", errF).Error())
	// 		fmt.Println(string(b))
	// 	}

	// 	t, err := template.ParseFiles("scane_template.html")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	buf := new(bytes.Buffer)
	// 	t.Execute(buf, worker.Openports)
	// 	sendMail(buf.String())
	// }

}
