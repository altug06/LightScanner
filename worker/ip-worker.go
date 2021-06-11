package worker

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ports       chan []byte
	Openports   map[string][]string
	serverCount int32
	Hosts       []HostInfo
)

type WorkerPool struct {
	Pool       chan chan DomIpPair
	JobQueue   chan DomIpPair
	MaxWorkers int
	workers    []*Worker
}

func NewPool(maxWorkers int, queue chan DomIpPair) *WorkerPool {
	if Openports == nil {
		Openports = make(map[string][]string)
	}
	pool := make(chan chan DomIpPair, maxWorkers)
	return &WorkerPool{Pool: pool, JobQueue: queue, MaxWorkers: maxWorkers}
}

func (p *WorkerPool) ShutDown() {
	for _, w := range p.workers {
		w.Stop()
	}
}

func LogFile(logFile string) {
	var openports *bufio.Writer

	openportslist, errC := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if errC != nil {
		log.Printf("Couldnt open/create the file: " + errC.Error())
	}

	openports = bufio.NewWriter(openportslist)

	for p := range ports {
		openports.Write(p)
		unflushedBufferSize := openports.Buffered()
		if unflushedBufferSize >= 4000 {
			errF := openports.Flush()
			if errF != nil {
				log.Printf("couldnt write buffered connection names to disk: " + errF.Error())
			}
		}
	}
}

func checkPort(port string) bool {
	whiteList := []string{"80", "8080", "443"}
	for _, p := range whiteList {
		if p == port {
			return true
		}
	}
	return false
}

func (p *WorkerPool) InitializeWorkers(wg *sync.WaitGroup, logFile string) {

	for i := 0; i < p.MaxWorkers; i++ {
		worker := NewWorker(p.Pool, wg, i+1)
		p.workers = append(p.workers, worker)
		worker.Start()
	}
	serverCount = 0

	ports = make(chan []byte)

	//go LogFile(logFile)
	go p.ExecuteQueue()
}

type DomIpPair struct {
	IP     string
	Domain []string
}

func (p *WorkerPool) ExecuteQueue() {
	for {
		select {
		case e := <-p.JobQueue:
			go func(e DomIpPair) {
				worker := <-p.Pool
				worker <- e
			}(e)
		}
	}
}

type Worker struct {
	WorkerPool  chan chan DomIpPair
	JobChannel  chan DomIpPair
	QuitChannel chan bool
	lock        Semaphore
	wg          *sync.WaitGroup
	ID          int
}

func (w *Worker) ScanPort(ip string, port int, timeout time.Duration) {
	target := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(timeout)
			w.ScanPort(ip, port, timeout)
		}
		return
	}

	conn.Close()
	// if !checkPort(port) {
	// 	p := strconv.Itoa(port)
	// 	fmt.Println(ip + ": " + p)
	// 	Openports[ip] = append(Openports[ip], p)
	// }
	p := strconv.Itoa(port)
	fmt.Println(ip + ": " + p)
	Openports[ip] = append(Openports[ip], p)

}

type MailBody struct {
	Body    string `json:"body"`
	Subject string `json:"subject"`
	To      string `json:"to"`
	Html    bool   `json:"html"`
}

type HostInfo struct {
	Ports  []string
	Domain []string
	IP     string
}

func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerPool <- w.JobChannel
			select {
			case e := <-w.JobChannel:
				if _, ok := Openports[e.IP]; !ok {
					fmt.Printf("worker-%d scans this ip: %s\n", w.ID, e.IP)
					var wg sync.WaitGroup
					for port := 1; port <= 65535; port++ {
						w.lock.Acquire(1)
						wg.Add(1)
						go func(port int) {
							defer w.lock.Release(1)
							defer wg.Done()
							w.ScanPort(e.IP, port, 500*time.Millisecond)
						}(port)
					}

					wg.Wait()
					atomic.AddInt32(&serverCount, 1)
					fmt.Println(serverCount)

					if len(Openports[e.IP]) != 0 {
						if len(Openports[e.IP]) <= 3 {
							i := 0
							for _, p := range Openports[e.IP] {
								if checkPort(p) {
									i++
								}
							}
							if len(Openports[e.IP]) != i {
								h := HostInfo{
									Ports:  Openports[e.IP],
									Domain: e.Domain,
									IP:     e.IP,
								}
								fmt.Println(h)
								Hosts = append(Hosts, h)

								fmt.Println("Open ports for Ip: " + e.IP)
								for port, _ := range Openports[e.IP] {
									fmt.Println(port)
								}
							}
						} else {
							h := HostInfo{
								Ports:  Openports[e.IP],
								Domain: e.Domain,
								IP:     e.IP,
							}
							fmt.Println(h)
							Hosts = append(Hosts, h)

							fmt.Println("Open ports for Ip: " + e.IP)
							for _, port := range Openports[e.IP] {
								fmt.Println(port)
							}
						}

					}

					w.wg.Done()

				}
			case <-w.QuitChannel:
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	go func() {
		w.QuitChannel <- true
	}()
}

func NewWorker(Pool chan chan DomIpPair, w *sync.WaitGroup, workerID int) *Worker {
	return &Worker{WorkerPool: Pool, JobChannel: make(chan DomIpPair), QuitChannel: make(chan bool), lock: make(Semaphore, 1024), wg: w, ID: workerID}
}
