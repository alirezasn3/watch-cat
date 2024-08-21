package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	goSystemd "github.com/alirezasn3/go-systemd"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"google.golang.org/protobuf/proto"
)

//go:embed public/build/*
var publicFS embed.FS

type Config struct {
	ListenAddress  string   `json:"listenAddress"`
	Destinations   []string `json:"destinations"`
	MonitorAddress string   `json:"monitorAddress"`
}

type ResultsQueue struct {
	Capacity int
	Results  []*PingResult
	Mu       sync.RWMutex
}

func (q *ResultsQueue) Append(r *PingResult) {
	if len(q.Results) == q.Capacity {
		q.Results = append(q.Results[1:], r)
	} else {
		q.Results = append(q.Results, r)
	}
}

func ping(dst string) {
	dstAddress := &net.IPAddr{IP: net.ParseIP(dst)}

	// create time map
	t := make(map[int]int64)

	// create mutex for time map
	l := &sync.Mutex{}

	// create packet connection
	c, e := icmp.ListenPacket("ip4:icmp", config.ListenAddress)
	if e != nil {
		panic(e)
	}
	defer c.Close()

	// send packets on seperate thread
	go func() {
		// create icmp message
		m := icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:  os.Getpid() & 0xffff,
				Seq: 1,
			},
		}

		var b []byte
		var e error
		for {
			// get icmp message bytes
			b, e = m.Marshal(nil)
			if e != nil {
				panic(e)
			}

			// send packet
			if _, e := c.WriteTo(b, dstAddress); e != nil {
				panic(e)
			}

			l.Lock()
			t[m.Body.(*icmp.Echo).Seq] = time.Now().UnixMilli()
			l.Unlock()

			// increment sequence
			m.Body.(*icmp.Echo).Seq++

			// sleep
			time.Sleep(time.Second)
		}
	}()

	// check for timed out packets
	go func() {
		var now int64
		for {
			now = time.Now().UnixMilli()
			l.Lock()
			for seq, r := range t {
				if now-r > 5000 {
					resultsQueue.Append(&PingResult{Destination: dst, RTT: -1, Seq: int32(seq), At: t[seq]})
					t[seq] = -1
				}
			}
			l.Unlock()
			time.Sleep(time.Second * 10)
		}
	}()

	// remove timed out packets from time map
	go func() {
		for {
			l.Lock()
			for seq, r := range t {
				if r < 0 {
					delete(t, seq)
				}
			}
			l.Unlock()
			time.Sleep(time.Minute)
		}
	}()

	// read packets
	b := make([]byte, 64)
	var n int
	var peer net.Addr
	var now int64
	var sentAt int64
	var seq int
	var m *icmp.Message
	pid := os.Getpid() & 0xffff
	for {
		n, peer, e = c.ReadFrom(b)
		now = time.Now().UnixMilli()
		if e != nil {
			panic(e)
		}

		// parse packet
		m, e = icmp.ParseMessage(1, b[:n]) // ipv4.ICMPTypeEchoReply.Protocol() replaced with 1 to improve performance
		if e != nil {
			panic(e)
		}

		if peer.String() == dst && m.Body.(*icmp.Echo).ID == pid && m.Type == ipv4.ICMPTypeEchoReply {
			seq = m.Body.(*icmp.Echo).Seq
			sentAt = t[seq]
			l.Lock()
			delete(t, seq)
			l.Unlock()
			if now-sentAt > 5000 {
				continue
			}
			resultsQueue.Mu.Lock()
			resultsQueue.Append(&PingResult{Destination: dst, RTT: now - sentAt, Seq: int32(seq), At: sentAt})
			resultsQueue.Mu.Unlock()
		}
	}
}

var config Config
var resultsQueue ResultsQueue

func init() {
	// Read config file
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	path := filepath.Dir(execPath)
	bytes, err := os.ReadFile(filepath.Join(path, "config.json"))
	if err != nil {
		panic(err)
	}
	// Parse config file into global config variable
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}
}

func main() {
	resultsQueue = ResultsQueue{Capacity: len(config.Destinations) * 500, Results: make([]*PingResult, 0, len(config.Destinations)*500)}

	for _, dst := range config.Destinations {
		go ping(dst)
	}

	if slices.Contains(os.Args, "--install") {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = goSystemd.CreateService(&goSystemd.Service{Name: "watchcat", ExecStart: execPath, Restart: "on-failure", RestartSec: "5s"})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("watchcat service created")
			os.Exit(0)
		}
	} else if slices.Contains(os.Args, "--uninstall") {
		err := goSystemd.DeleteService("watchcat")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("watchcat service deleted")
			os.Exit(0)
		}
	}

	e := echo.New()

	e.Use(middleware.CORS())

	if len(os.Args) > 1 && slices.Contains(os.Args, "--live") {
		log.Print("using live mode")
		execPath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		path := filepath.Dir(execPath)
		e.Static("/", filepath.Join(path, "public", "build"))
	} else {
		log.Print("using embed mode")
		e.StaticFS("/*", echo.MustSubFS(publicFS, "public/build"))
	}

	e.GET("/api/results", func(c echo.Context) error {
		resultsQueue.Mu.RLock()
		defer resultsQueue.Mu.RUnlock()
		b, err := proto.Marshal(&PingResults{Results: resultsQueue.Results})
		if err != nil {
			e.Logger.Error(err)
			return c.NoContent(500)
		}
		return c.Blob(200, "application/x-protobuf", b)
	})

	e.Logger.Fatal(e.Start(config.MonitorAddress))
}

// func (r *PingResult) serialize() []byte {
// 	loc, e := time.LoadLocation("Iran")
// 	if e != nil {
// 		panic(e)
// 	}
// 	fmt.Println(time.Now().In(loc).Format("2006-01-02-15-04-05"))
// 	// Create a new buffer to write the serialized data to
// 	var b bytes.Buffer
// 	// Create a new gob encoder and use it to encode the person struct
// 	enc := gob.NewEncoder(&b)
// 	if err := enc.Encode(r); err != nil {
// 		fmt.Println("Error encoding struct:", err)
// 		return nil
// 	}
// 	// The serialized data can now be found in the buffer
// 	return b.Bytes()
// }
// func (r *PingResult) deserialize(data []byte) {
// 	b := bytes.NewBuffer(data)
// 	// Create a new gob decoder and use it to decode the person struct
// 	dec := gob.NewDecoder(b)
// 	if err := dec.Decode(r); err != nil {
// 		fmt.Println("Error decoding struct:", err)
// 	}
// }
