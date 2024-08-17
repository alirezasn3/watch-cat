package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Config struct {
	ListenAddress  string   `json:"listenAddress"`
	Destinations   []string `json:"destinations"`
	MonitorAddress string   `json:"monitorAddress"`
}

type PingResult struct {
	Destination string `json:"destination"`
	RTT         int64  `json:"rtt"`
	Seq         int    `json:"seq"`
	At          int64  `json:"at"`
}

type PingResults struct {
	results []PingResult
	mu      sync.RWMutex
}

func (r *PingResult) serialize() []byte {
	loc, e := time.LoadLocation("Iran")
	if e != nil {
		panic(e)
	}
	fmt.Println(time.Now().In(loc).Format("2006-01-02-15-04-05"))

	// Create a new buffer to write the serialized data to
	var b bytes.Buffer
	// Create a new gob encoder and use it to encode the person struct
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(r); err != nil {
		fmt.Println("Error encoding struct:", err)
		return nil
	}
	// The serialized data can now be found in the buffer
	return b.Bytes()
}

func (r *PingResult) deserialize(data []byte) {
	b := bytes.NewBuffer(data)
	// Create a new gob decoder and use it to decode the person struct
	dec := gob.NewDecoder(b)
	if err := dec.Decode(r); err != nil {
		fmt.Println("Error decoding struct:", err)
	}
}

func ping(dst string) {
	dstAddress := &net.IPAddr{IP: net.ParseIP(dst)}

	// create time map
	t := make(map[int]int64)

	// create mutex for time map
	l := &sync.RWMutex{}

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

	// read packets
	b := make([]byte, 64)
	var n int
	var peer net.Addr
	var now int64
	for {
		n, peer, e = c.ReadFrom(b)
		now = time.Now().UnixMilli()
		if e != nil {
			panic(e)
		}

		// parse packet
		m, e := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), b[:n])
		if e != nil {
			panic(e)
		}

		if peer.String() == dst && m.Body.(*icmp.Echo).ID == os.Getpid()&0xffff && m.Type == ipv4.ICMPTypeEchoReply {
			// fmt.Printf("%d bytes from %s: icmp_seq=%d time=%d ms\n", n, dst, m.Body.(*icmp.Echo).Seq, now-t[m.Body.(*icmp.Echo).Seq])
			pingResults.mu.Lock()
			l.RLock()
			pingResults.results = append(pingResults.results, PingResult{Destination: dst, RTT: now - t[m.Body.(*icmp.Echo).Seq], Seq: m.Body.(*icmp.Echo).Seq, At: t[m.Body.(*icmp.Echo).Seq]})
			l.RUnlock()
			pingResults.mu.Unlock()
		}
	}
}

var config Config
var pingResults PingResults

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
	for _, dst := range config.Destinations {
		go ping(dst)
	}
	http.ListenAndServe(config.MonitorAddress, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pingResults.mu.RLock()
		b, e := json.Marshal(pingResults.results)
		pingResults.mu.RUnlock()
		if e != nil {
			fmt.Println(e)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Write(b)
	}))
}
