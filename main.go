package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
	"strings"

	//"common"
)
const MAX_BUFF int = 100
const BUFF_SIZE int = 512

type SafeInt struct {
	sync.Mutex
	Num int
}

var g_pos SafeInt
var gbuffer [MAX_BUFF][]byte
var gmap map[string]int
var g_adwords map[string]int
var server_list []string
var server_ind int

func main() {
	host := flag.String("host", "localhost", "interface to listen on")
	port := flag.Int("port", 5353, "dns port to listen on")
	dohserver := flag.String("dohserver", "https://mozilla.cloudflare-dns.com/dns-query", "DNS Over HTTPS server address")
	domserver := flag.String("innserver", "180.76.76.76,114.114.114.114,223.5.5.5,119.29.29.29", "Domestic Dns server address")
	debug := flag.Bool("debug", false, "print debug logs")
	chn_file := flag.String("chn", "cn.txt", "default domestic domain list file")
	block_file := flag.String("block", "block.txt", "default ad keyword list file")
	flag.Parse()

	gmap = get_config(*chn_file,true)
	g_adwords = get_config(*block_file,false)

	if *debug {
		log.SetFlags(log.Lshortfile)
	} else {
		log.SetFlags(0)
	}

	g_pos.Num = 0 

	server_list = strings.Split(*domserver,",")
	server_ind = 0

	for i := 0; i < len(server_list); i++ {
    	log.Printf("dns server : %s", server_list[i])
    }
    
    for i := 0; i < MAX_BUFF; i++ {
    	gbuffer[i] = make([]byte, BUFF_SIZE)
    }



	if err := newUDPServer(*host, *port, *dohserver); err != nil {
		log.Fatalf("could not listen on %s:%d: %s", *host, *port, err)
	}
}



func newUDPServer(host string, port int, dohserver string) error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		return err
	}
	for {
		//var raw [512]byte
		raw := get_next_buff()
		n, addr, err := conn.ReadFromUDP(raw)
		if err != nil {
			log.Printf("could not read: %s", err)
			continue
		}
		//log.Printf("new connection from %s:%d", addr.IP.String(), addr.Port)
		
		url := get_url(raw[13:])
        if len(url) == 0 {
        	continue
        }

        if raw[n-3] == 28 {
    	//do not support ipv6 request
    	continue 
       }

		if is_blocked(url, g_adwords) == true {
           log.Printf("blocked  : %s ", url)
           continue
		} else {

	          if is_chn_domain(url,gmap) == true {
	             log.Printf("domestic : %s ", url)
	             go domestic_query(get_next_server(), conn, addr, raw[:n])
	          }else{
	          	 log.Printf("oversea  : %s ", url)
	          	go proxy(dohserver, conn, addr, raw[:n])
	          }
	    }
	} //end for
}

func domestic_query(domserver string, conn *net.UDPConn, Remoteaddr *net.UDPAddr, raw []byte) {

    //log.Printf("%v", raw)
    nstr := domserver
	if strings.Contains(domserver,":") == false {
       nstr += ":53"
	}

	addr, err := net.ResolveUDPAddr("udp", nstr)
	if err != nil {
		log.Printf("Can't resolve address: %v", err)
		return
	}

	cliConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Can't dial: ", err)
		return
	}
	defer cliConn.Close()

	// todo set timeout
	_, err = cliConn.Write(raw)
	remoteBuf := get_next_buff()

	cliConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = cliConn.Read(remoteBuf)
	if err != nil {
		log.Printf("read udp fail: %v\n", err)
		return
	}

	conn.WriteToUDP(remoteBuf,Remoteaddr)
	return 
}



func proxy(dohserver string, conn *net.UDPConn, addr *net.UDPAddr, raw []byte) {
	enc := base64.RawURLEncoding.EncodeToString(raw)
	url := fmt.Sprintf("%s?dns=%s", dohserver, enc)
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("could not create request: %s", err)
		return
	}
	r.Header.Set("Content-Type", "application/dns-message")
	r.Header.Set("Accept", "application/dns-message")

	c := http.Client{}
	resp, err := c.Do(r)
	if err != nil {
		log.Printf("could not perform request: %s", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("wrong response from DOH server got %s", http.StatusText(resp.StatusCode))
		return
	}

	msg, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("could not read message from response: %s", err)
		return
	}

	if _, err := conn.WriteToUDP(msg, addr); err != nil {
		log.Printf("could not write to udp connection: %s", err)
		return
	}
}

func get_next_buff() []byte {

    g_pos.Lock()
    //log.Printf("url = %s, buffer pos = %d\n", url,g_pos.Num)
    old_pos := g_pos.Num
			g_pos.Num += 1 
            if g_pos.Num == MAX_BUFF {
               g_pos.Num = 0

            }

       g_pos.Unlock()
     return gbuffer[old_pos]
}

func get_next_server() string {
	max  := len(server_list)
    old_pos := server_ind
    server_ind += 1
    if server_ind == max{
    	server_ind = 0
    }

    return server_list[old_pos]
}


