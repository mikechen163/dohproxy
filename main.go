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
const MAX_BUFF int = 300
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
var   default_ttl  float64 

var oversea_server_list []string
var oversea_server_ind int

type DnsCache struct {
	ttl  time.Time
	req_type byte
	msg []byte
}

var rwLock *sync.RWMutex 
var g_buffer map[string]DnsCache

func main() {
	host := flag.String("host", "localhost", "interface to listen on")
	port := flag.Int("port", 5353, "dns port to listen on")
	ttl := flag.Int("ttl", 7200, "default oversea ttl length")
	//dohserver := flag.String("dohserver", "https://mozilla.cloudflare-dns.com/dns-query", "DNS Over HTTPS server address")
	dohserver := flag.String("dohserver", "https://8.8.8.8/dns-query", "DNS Over HTTPS server address")
	
	domserver := flag.String("innserver", "223.5.5.5,119.29.29.29", "Domestic Dns server address")
	debug := flag.Bool("debug", false, "print debug logs")
	fallback_mode := flag.Bool("fallback", false, "set fallback mode")
	cache_enabled:= flag.Bool("cached", true, "set cache mode : experiment!")
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

    	log.Printf("listen on port : %d", *port)
	
	server_list = strings.Split(*domserver,",")
	server_ind = 0

	for i := 0; i < len(server_list); i++ {
    	log.Printf("domestic dns server : %s", server_list[i])
    }


    //log.Printf("over dns server : %s", *dohserver)
  
    if strings.Contains(*dohserver,",") == true {

		oversea_server_list = strings.Split(*dohserver,",")
		oversea_server_ind = 0

		for i := 0; i < len(oversea_server_list); i++ {
	    	log.Printf("oversea dns server : %s", oversea_server_list[i])
	    }

    }else {
        log.Printf("oversea dns server : %s", *dohserver)  	
        oversea_server_list = make([]string, 1)
        oversea_server_list[0] = *dohserver
        oversea_server_ind = 0
    }


    if (*fallback_mode == true ) {
     log.Printf("fallback is true")
    }
    
    for i := 0; i < MAX_BUFF; i++ {
    	gbuffer[i] = make([]byte, BUFF_SIZE)
    }

     rwLock = new(sync.RWMutex)
     g_buffer = make(map[string]DnsCache)
     default_ttl = float64(*ttl)


	if err := newUDPServer(*host, *port, *dohserver ,*fallback_mode , *cache_enabled); err != nil {
		log.Fatalf("could not listen on %s:%d: %s", *host, *port, err)
	}
}



func newUDPServer(host string, port int, dohserver string, fallback_mode bool , cache_enabled bool) error {
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
		
		url := get_url(raw[12:])
        if len(url) == 0 {
        	continue
        }

    
        req_type := raw[len(url)+12+3]

        //  if req_type == 12 {
        // 	continue
        // }

       // log.Printf("new connection from %s:%d , %s %d ", addr.IP.String(), addr.Port,url,req_type)

        if cache_enabled == true {

	        if cache, ok := read_map(get_key(url,req_type)); ok {
	          
			  du := time.Since(cache.ttl).Seconds() 
			  if du <=  default_ttl {

			  	  log.Printf("cached found : %s", url)
	              //update tid
	              cache.msg[0] = raw[0]
	              cache.msg[1] = raw[1]

				  if _, err := conn.WriteToUDP(cache.msg, addr); err != nil {
				    log.Printf("could not write cache to local udp connection: %s", err)
				  }
			  } //ttl valie
		    }  // cached found

	   } //end of cache_enable

        //if raw[n-3] == 28 {
    	//do not support ipv6 request
    	//continue 
       //}

        if strings.HasSuffix(url,".lan") {
    	// .lan
    	continue 
       }

		if is_blocked(url, g_adwords) == true {
           log.Printf("blocked  : %s ", url)
           continue
		} else {

	          if ((is_chn_domain(url,gmap) == true) || (fallback_mode == true)) {
	             //log.Printf("req_type %02d , domestic : %s ", req_type ,url)
	                         
	             for i := 0; i < len(server_list); i++ {
	             	go domestic_query(get_next_server(), conn, addr, raw[:n] , false)
    	         }

	          }else{


	          	 //log.Printf("req_type %02d , oversea  : %s ", req_type,url)
			 
                	if strings.Contains(dohserver,"https") == true {
	                	go proxy(dohserver, conn, addr, raw[:n] , cache_enabled)
			        }else{
			          go domestic_query(get_next_oversea_server(), conn, addr, raw[:n] , cache_enabled)
	         	   }

	          }
	    }
	} //end for
}

func domestic_query(domserver string, conn *net.UDPConn, Remoteaddr *net.UDPAddr, raw []byte , cache_flag bool) {

    //log.Printf("%v", raw)
    nstr := domserver
	if strings.Contains(domserver,":") == false {
       nstr += ":53"
	}

	log.Printf("send udp request to %s %s", nstr , get_url(raw[12:]))

	addr, err := net.ResolveUDPAddr("udp", nstr)
	if err != nil {
		log.Printf("Can't resolve address: %s %v", nstr, err)
		return
	}

	cliConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Printf("Can't dial: %s %v", nstr,err)
		return
	}
	defer cliConn.Close()

	// todo set timeout
	_, err = cliConn.Write(raw)
	remoteBuf := get_next_buff()

	cliConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = cliConn.Read(remoteBuf)
	if err != nil {
		log.Printf("read remote dns server fail: %s %v\n", nstr,err)
		return
	}

	if _, err := conn.WriteToUDP(remoteBuf, Remoteaddr); err != nil {
		log.Printf("could not write to local connection: %v", err)
		return
	}

	//log.Printf("Receive succ resp from : %s %s", nstr , get_url(raw[12:]))

	if cache_flag == true {

		url := get_url(raw[12:])
		req_type := raw[len(url)+12+3]

		if _ , ok := read_map(get_key(url,req_type)); ok {	
			delete_map(get_key(url,req_type))
	     }		

	    add_node(remoteBuf,url,req_type)
   } //end of cache_flag

}



func proxy(dohserver string, conn *net.UDPConn, addr *net.UDPAddr, raw []byte , cache_enabled bool) {
	enc := base64.RawURLEncoding.EncodeToString(raw)
	url := fmt.Sprintf("%s?dns=%s", dohserver, enc)

	log.Printf("send https request to %s %s", dohserver , get_url(raw[12:]))

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


    if cache_enabled == true {
    
		url = get_url(raw[12:])
		req_type := raw[len(url)+12+3]

        if _ , ok := read_map(get_key(url,req_type)); ok {	
			delete_map(get_key(url,req_type))
	     }		

	    add_node(msg,url,req_type)
		
    }  

}

func read_map(key string) (DnsCache ,bool){

    rwLock.RLock()
    value, ok := g_buffer[key]
    rwLock.RUnlock()

    return value,ok
}

func write_map(key string, ele DnsCache){
  rwLock.Lock()
  g_buffer[key] = ele
  rwLock.Unlock()
}

func delete_map(key string){
  rwLock.Lock()
  delete(g_buffer,key)
  rwLock.Unlock()
}

func get_key(url string,req_type byte) string{
	return url + string(req_type)
}

func add_node(msg []byte, url string, req_type byte){
	    var ele DnsCache

        ele.msg = make([]byte, len(msg))
        copy(ele.msg, msg) 

		ele.ttl = time.Now()
		ele.req_type = req_type
		
       // g_buffer[get_key(url,req_type)] = ele
       write_map(get_key(url,req_type),ele)

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

func get_next_oversea_server() string {
	max  := len(oversea_server_list)
    old_pos := oversea_server_ind
    oversea_server_ind += 1
    if oversea_server_ind == max{
    	oversea_server_ind = 0
    }

    return oversea_server_list[old_pos]
}

 func deepCopy(s string) string {
     var sb strings.Builder
     sb.WriteString(s)
     return sb.String()
 }

