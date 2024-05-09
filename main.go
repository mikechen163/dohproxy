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
	"strconv"
	"encoding/binary"

	//"common"
)
const MAX_BUFF int = 1000
const BUFF_SIZE int = 512
//const KEEP_ALIVE_TIME_LEN int = 30
const KEEP_ALIVE_TIME_LEN time.Duration = 30


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
	//timer *time.Timer
	ttl  time.Time
	req_type byte
	msg []byte
}

var rwLock *sync.RWMutex 
var tcpLock *sync.RWMutex 
var g_buffer map[string]DnsCache
var g_cache_timer *time.Ticker
var g_edns_subnet string


type TcpConnPool struct {
	timer *time.Timer
	ticker *time.Ticker
	conn *net.TCPConn
}

type UdpConnPool struct {
	conn *net.UDPConn
	addr *net.UDPAddr
	url string
	req_type byte
}

var g_tcp_conn_pool map[string]TcpConnPool
var g_dns_context_id map[uint16]UdpConnPool
var tcp_reuse_flag bool
var ipv6_oversea_flag bool
var g_google_buf []byte

func main() {
	host := flag.String("host", "localhost", "interface to listen on")
	port := flag.Int("port", 5353, "dns port to listen on")
	ttl := flag.Int("ttl", 7200, "default oversea ttl length")
	//dohserver := flag.String("dohserver", "https://mozilla.cloudflare-dns.com/dns-query", "DNS Over HTTPS server address")
	dohserver := flag.String("dohserver", "https://8.8.8.8/dns-query", "DNS Over HTTPS server address")
	
	domserver := flag.String("innserver", "223.5.5.5,119.29.29.29", "Domestic Dns server address")
	edns_subnet := flag.String("subnet", "0.0.0.0", "set edns default subnet")
	debug := flag.Bool("debug", false, "print debug logs")
	fallback_mode := flag.Bool("fallback", false, "set fallback mode")
	cache_enabled := flag.Bool("cached", true, "set cache mode : experiment!")
	tcp_reuse := flag.Bool("tcp_reuse", true, "dns over tcp and tcp_reuse_flag : experiment!")
	ipv6_oversea := flag.Bool("ipv6_oversea", false, "disable ipv6 for oversea domain, default disable : experiment!")
	chn_file := flag.String("chn", "cn.txt", "default domestic domain list file")
	block_file := flag.String("block", "block.txt", "default ad keyword list file")
	
	flag.Parse()

	gmap = get_config(*chn_file,true)
	g_adwords = get_config(*block_file,false)

	g_google_buf = []byte{0x18,0x6f,0x01,0x20,0x00,0x01,0x00,0x00,0x00,0x00,0x00,0x01,0x03,0x77,0x77,0x77,0x06,0x67,0x6f,0x6f,0x67,0x6c,0x65,0x03,0x63,0x6f,0x6d,0x00,0x00,0x01,0x00,0x01,0x00,0x00,0x29,0x10,0x00,0x00,0x00,0x00,0x00,0x00,0x00}


  

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

    g_tcp_conn_pool = make(map[string]TcpConnPool)
    g_dns_context_id = make(map[uint16]UdpConnPool)
    g_edns_subnet = *edns_subnet


    if (*fallback_mode == true ) {
     log.Printf("fallback is true")
    }
    
    for i := 0; i < MAX_BUFF; i++ {
    	gbuffer[i] = make([]byte, BUFF_SIZE)
    }

     rwLock = new(sync.RWMutex)
     tcpLock = new(sync.RWMutex)
     g_buffer = make(map[string]DnsCache)
     default_ttl = float64(*ttl)

     dur := time.Duration(int(default_ttl))
	 g_cache_timer = time.NewTicker(dur * time.Second)

	 go start_cache_timer(g_cache_timer)

    

      
     if   *tcp_reuse {
     	tcp_reuse_flag = true
     }
     
     if   *ipv6_oversea {
     	ipv6_oversea_flag = true
     }
    

	if err := newUDPServer(*host, *port, *dohserver ,*fallback_mode , *cache_enabled); err != nil {
		log.Fatalf("could not listen on %s:%d: %s", *host, *port, err)
	}
}

func append_edns_subnet(raw []byte, n int)( int ) {
  
  //default do nothing
  if (g_edns_subnet == "0.0.0.0"){
  	return n
  }

  
  //convert from string to int
  tr := strings.Split(g_edns_subnet,".")

  buf := []byte{0x00,0x00,0x29,0x10,0x00,0x00,0x00,0x00,0x00,0x00,0x0b,0x00,0x08,0x00,0x07,0x00,0x01,0x18,0x00,0x67,0x15,0xc7}
  buflen := len(buf)

  
  for i, v := range tr {
  	t , _ := strconv.Atoi(v)

  	if (i < 3) {
  	  buf[buflen-3+i] = byte(t)
    }
  }



  url := get_url(raw[12:])
  npos := len(url) + 12 + 6

  //dns rr number = 0 
  if (raw[11] == 0) {
    copy(raw[n:],buf)
    n += len(buf)
    raw[11] = 1
    return n
  }

  
  if ((raw[11] == 1) && (raw[n-1] == 0)) {
  copy(raw[npos:],buf)
  n = npos + len(buf)
  raw[11] = 1
  return n
  }

  //rr > 2 
  //do nothing
  return n 

}

func print_buf( raw []byte, size int){

      
        fmt.Printf("\n") 

	      i := 0
		 for _, nx := range(raw[:size]) {
          i += 1
          fmt.Printf("0x%02x,", nx) 
         
          if (i == 16){
          	i = 0
           fmt.Printf("\n")
          }

         }

           fmt.Printf("\n") 

        
}

func newUDPServer(host string, port int, dohserver string, fallback_mode bool , cache_enabled bool) error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host), Port: port})

     //ipv6_null_message := []byte{ 0x00, 0x01, 0x00, 0x00, 0x00, 0x78, 0x00, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
                
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
		
		//print_buf(raw,n)
		
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

                 // cache.timer.Stop()
				 // delete_map(get_key(url,req_type))
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

                 if (req_type == 28) && (ipv6_oversea_flag == false) {
    	         //do not support ipv6 request for oversea
 
                 raw[2] = 0x85
                 raw[3] = 0x80
                 //raw[7] = 0x01
                 //copy(raw[(len(url)+12+6):], raw[12:(len(url)+16)])
                 //copy(raw[(len(url)+6+(len(url)+16)):], ipv6_null_message)
                 //log.Printf("null ipv6\n")
                 //print_buf(raw,len(url)+12+(len(url)+16)+18)

                  if _, err := conn.WriteToUDP(raw[:n], addr); err != nil {
				  //if _, err := conn.WriteToUDP(raw[:(len(url)+12+(len(url)+16)+18)], addr); err != nil {
				    log.Printf("could not write ipv6 deny message to local udp connection: %s", err)
				  }

    	         continue 
                 }


	          	  n := append_edns_subnet(raw,n)
	          	  //print_buf(raw,n)
	          	  //return nil

	          	 //log.Printf("req_type %02d , oversea  : %s ", req_type,url)
			 
                	if strings.Contains(dohserver,"https") == true {
	                	go proxy(dohserver, conn, addr, raw[:n] , cache_enabled)
			        }else{

			          if strings.Contains(dohserver,"tcp") == true {
			          	go tcp_query(get_next_oversea_server(), conn, addr, raw[:n] , cache_enabled)
			          }else{
			            go domestic_query(get_next_oversea_server(), conn, addr, raw[:n] , cache_enabled)
			          }
	         	   }

	          }
	    }
	} //end for
}

//find any available tcp connection in pool
func get_available_conn(nstr string)(TcpConnPool, bool) {

	var tt TcpConnPool 

	//log.Printf("get_tcp_conn,tcp_pool_size = %d, dns_context_size = %d ", len(g_tcp_conn_pool), len(g_dns_context_id))
     
    tcpLock.Lock()
	for _, v := range g_tcp_conn_pool {
		//log.Printf("traval all map k = %s , v = %s , nstr = %s  ",  k, v.conn.RemoteAddr().String(), nstr)
        if (nstr == v.conn.RemoteAddr().String()) {
        	tcpLock.Unlock()
        	return v,true
        }
  
    }

    tcpLock.Unlock()
    return tt,false
}

func get_tcpconn_key(conn *net.TCPConn)(string){
	return conn.LocalAddr().String() + conn.RemoteAddr().String()
}

func reset_tcp_conn( conn *net.TCPConn){

	    //ele := g_tcp_conn_pool[get_tcpconn_key(conn)]

        if (false == tcp_reuse_flag) {
           return
        }

        tcpLock.Lock()
        delete(g_tcp_conn_pool,get_tcpconn_key(conn))
        tcpLock.Unlock()

        //conn.Close()
        //ele.ticker.Stop()	
}

func start_timer(timer *time.Timer ,ticker *time.Ticker , conn *net.TCPConn){
         <- timer.C
    	
        log.Printf("Timer out , release socket: %s <-> %s ",  conn.LocalAddr().String(), conn.RemoteAddr().String())
        
        reset_tcp_conn(conn)
        conn.Close()

        //log.Printf("tcp_pool_size = %d, dns_context_size = %d ", len(g_tcp_conn_pool), len(g_dns_context_id))
        
        //clear g_dns_context_id for resource check
        if ( (len(g_tcp_conn_pool) == 0 ) && ( len(g_dns_context_id) > 100 ) ) {

        	log.Printf("reset dns_context_map,length = %d ",  len(g_dns_context_id))
        	tcpLock.Lock()
            g_dns_context_id = make(map[uint16]UdpConnPool)
            tcpLock.Unlock()
           
        }

        ticker.Stop()
}

func keep_alive(ticker *time.Ticker,conn *net.TCPConn) {

    var counter uint16

    counter = 0;

	for {
	  <- ticker.C
      //log.Printf("Ticker , keep_alive socket: %s <-> %s ",  conn.LocalAddr().String(), conn.RemoteAddr().String())
      buf := []byte{0x18,0x6f,0x01,0x20,0x00,0x01,0x00,0x00,0x00,0x00,0x00,0x01,0x03,0x77,0x77,0x77,0x06,0x67,0x6f,0x6f,0x67,0x6c,0x65,0x03,0x63,0x6f,0x6d,0x00,0x00,0x01,0x00,0x01,0x00,0x00,0x29,0x10,0x00,0x00,0x00,0x00,0x00,0x00,0x00}
      
      counter += 1
      binary.BigEndian.PutUint16(buf, counter)
      tcp_query("tcp://" + conn.RemoteAddr().String(), nil, nil,buf, false)

	}//end for

}

func start_cache_timer(ticker *time.Ticker){
        
 for {
	  <- ticker.C
      
     for k, v := range g_buffer {

     	 du := time.Since(v.ttl).Seconds() 
		 if du >  default_ttl {
		 	//log.Printf("Resource check , release cache: %s ",  k)
		 	delete_map(k)
		 }
     }
	}//end for
}        

func get_conn(domserver string, conn *net.UDPConn) (*net.TCPConn) {

  var ele TcpConnPool

   nstr := domserver[6:]
   nele , ok := get_available_conn(nstr)
  if (ok){
  	 //log.Printf("reuse socket: %s <-> %s ",  nele.conn.LocalAddr().String(), nele.conn.RemoteAddr().String())
     
     if (conn != nil){
      nele.timer.Reset(KEEP_ALIVE_TIME_LEN * time.Second)
     }

    return nele.conn 
  }else {
   
    //keep alive ,dont create new sock
    if (conn == nil) {
    	return nil
    } 

    addr, err := net.ResolveTCPAddr("tcp", nstr)
	if err != nil {
		log.Printf("Can't resolve address: %s %v", nstr, err)
		return nil
	}

	cliConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Printf("Can't dial: %s %v", nstr,err)
		return nil
	}

	ele.timer = time.NewTimer(KEEP_ALIVE_TIME_LEN * time.Second)
	ele.ticker = time.NewTicker(time.Second)
    ele.conn = cliConn

    tcpLock.Lock()
    g_tcp_conn_pool[get_tcpconn_key(ele.conn)] = ele
    tcpLock.Unlock()
    //tcpLock.Unlock()

    //log.Printf("create socket: %s <-> %s ",  ele.conn.LocalAddr().String(), ele.conn.RemoteAddr().String())

    go start_timer(ele.timer,ele.ticker,cliConn)
    go keep_alive(ele.ticker,cliConn)
   	return cliConn
 
  } //else

  return nil

}

func handle_dns_response(buf []byte , tag uint16, cliConn *net.TCPConn,conn *net.UDPConn, Remoteaddr *net.UDPAddr, url string, req_type byte, cache_flag bool){

	tag2  := binary.BigEndian.Uint16(buf[:2])
    //log.Printf("context id : %d",tag2)
   
    
    // packet return not in fifo mode. out-of-order. 
    if (tag != tag2) {
        tcpLock.Lock()
    	ra , ok := g_dns_context_id[tag2]
    	tcpLock.Unlock()

    	//found context, normal case, keep alive packet will return not found.
    	if (ok) {

    		tcpLock.Lock()
		     delete (g_dns_context_id,tag2)
		     tcpLock.Unlock()

    		if _, err := ra.conn.WriteToUDP(buf, ra.addr); err != nil {
			  log.Printf("could not write to local connection: %v", err)
			  return
		    }

		    log.Printf("Out-order dns success: %s->%s | %s\n", cliConn.LocalAddr().String(),cliConn.RemoteAddr().String(),url)

		    if cache_flag == true {

		    	//Answer RR number is not zero
				if ( binary.BigEndian.Uint16(buf[6:8]) == 0){
					return
				}

		    	 url = ra.url
		    	 req_type = ra.req_type
				
				if _ , ok := read_map(get_key(url,req_type)); ok {	
					delete_map(get_key(url,req_type))
			     }		

			    add_node(buf,url,req_type)
		    } //end of cache_flag

            
	    }else{

	    	//maybe a keep alive response, or context not found mode

	        //keep alive connection
    		//log.Printf("Out-order dns no context found: %s->%s | %s\n", cliConn.LocalAddr().String(),cliConn.RemoteAddr().String(),url)
	    }
    	return
    } // end of tag1 != tag2

    //this is the normal case, fifo module
    
    //keep alive test packet, do nothing
    if (conn == nil) {
    	return
    }

    tcpLock.Lock()
    delete (g_dns_context_id,tag)
    tcpLock.Unlock()

	if _, err := conn.WriteToUDP(buf, Remoteaddr); err != nil {
		log.Printf("could not write to local connection: %v", err)
		return
	}

    log.Printf("Normal dns success: %s->%s | %s\n", cliConn.LocalAddr().String(),cliConn.RemoteAddr().String(),url)

	if cache_flag == true {

		//Answer RR number is not zero
		if ( binary.BigEndian.Uint16(buf[6:8]) == 0){
			return
		} 

		if _ , ok := read_map(get_key(url,req_type)); ok {	
			delete_map(get_key(url,req_type))
	     }		

	    add_node(buf,url,req_type)
   } //end of cache_flag
  
}

func tcp_query(domserver string, conn *net.UDPConn, Remoteaddr *net.UDPAddr, raw []byte , cache_flag bool) {

    
    var cliConn *net.TCPConn
    var err error
    var step uint16
    var ele UdpConnPool
    var remoteBuf []byte
    var size int

    //log.Printf("%v", raw)
    nstr := domserver
	if strings.Contains(domserver,"tcp") == false {
       return 
	}

	//log.Printf("send tcp request to %s %s", nstr , get_url(raw[12:]))

    if (false == tcp_reuse_flag) {
		addr, err := net.ResolveTCPAddr("tcp", nstr[6:])
		if err != nil {
			log.Printf("Can't resolve address: %s %v", nstr[6:], err)
			return
		}

		cliConn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Printf("Can't dial: %s %v", nstr,err)
			return
		}
		defer cliConn.Close()
    }else{

       cliConn = get_conn(domserver,conn)
	   if (cliConn == nil){
         
         //keep alive case, just return
	   	 if (conn == nil){
	   	 	return
	   	 }

		 log.Printf("get_conn fail: %s ", domserver)
		 return
	   }

    }

    		
	tag  := binary.BigEndian.Uint16(raw[:2])
	
	if (conn != nil) {
		ele.conn = conn
		ele.addr = Remoteaddr
		ele.url = get_url(raw[12:])
		ele.req_type = raw[len(ele.url)+12+3]

		tcpLock.Lock()
		g_dns_context_id[tag] = ele
		tcpLock.Unlock()
    }
	//log.Printf("context id : %d",tag)

	nb := make([]byte, len(raw)+2)
	binary.BigEndian.PutUint16(nb, uint16(len(raw)))
	copy(nb[2:], raw)

	_, err = cliConn.Write(nb)
	if err != nil {
       log.Printf("write to server fail: %s %v", nstr,err)
        reset_tcp_conn(cliConn)
		return
    }

    //keep alive test packet, do nothing
    // if (conn == nil) {
    // 	return
    // }

	temp_buf := get_next_buff()

	cliConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	size_t, err := cliConn.Read(temp_buf)
	if err != nil {
		log.Printf("read remote dns server fail: %s %v\n", nstr,err)
        reset_tcp_conn(cliConn)
		return
	}

	remoteBuf = temp_buf
	size = size_t

	if (BUFF_SIZE == size) {
		    log.Printf("Rsp size bigger than 512: %s->%s | %s\n", cliConn.LocalAddr().String(),cliConn.RemoteAddr().String(),get_url(raw[12:]))
          
            buf_t := make([]byte,3*BUFF_SIZE) 
            //copy(remoteBuf,temp_buf)

	        size_t, err := cliConn.Read(buf_t)
		    if err != nil {
				log.Printf("read remote dns server fail: %s %v\n", nstr,err)
		        reset_tcp_conn(cliConn)
				return
		    }

		    if (size_t == 3*BUFF_SIZE){
		    	log.Printf("!!!!too big packet, can't handle size = %d \n" , size_t + BUFF_SIZE )
	            reset_tcp_conn(cliConn)
			    return
		    }

		    remoteBuf = make([]byte,4*BUFF_SIZE) 
		    copy(remoteBuf,temp_buf)
		    copy(remoteBuf[BUFF_SIZE:],buf_t)

		    size = size_t + BUFF_SIZE
	}

	
   
    step = 0
    for {
    	
	    size_2 := binary.BigEndian.Uint16(remoteBuf[(step):(2+step)])

	    if (size < int(size_2 + 2 + step)) {
	    	return
	    }
        
        //log.Printf("buf_size = %d, rsp size = %d, step = %d , buf=[%d:%d]",size,size_2,step,2+step,size_2 + 2 + step)
	    url := get_url(raw[12:])
		req_type := raw[len(url)+12+3]
   	    handle_dns_response(remoteBuf[(2+step):(size_2 + 2 + step)] , tag , cliConn,conn,Remoteaddr,url,req_type,cache_flag )

	   if (size <= int(size_2 + 2 + step)) {
	    	return
	    }

	   step += size_2 + 2
    }
    return
}

func domestic_query(domserver string, conn *net.UDPConn, Remoteaddr *net.UDPAddr, raw []byte , cache_flag bool) {

    //log.Printf("%v", raw)
    nstr := domserver
	if strings.Contains(domserver,":") == false {
       nstr += ":53"
	}

	//log.Printf("send udp request to %s %s", nstr , get_url(raw[12:]))

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
	n, err := cliConn.Read(remoteBuf)
	if err != nil {
		log.Printf("read remote dns server fail: %s %v\n", nstr,err)
		return
	}

	 //print_buf(remoteBuf,n)

	if _, err := conn.WriteToUDP(remoteBuf[:n], Remoteaddr); err != nil {
		log.Printf("could not write to local connection: %v", err)
		return
	}

	
	//log.Printf("udp succ rsp from %s | %s", nstr , get_url(raw[12:]))

	if cache_flag == true {

		log.Printf("udp succ rsp from %s | %s", nstr , get_url(raw[12:]))

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

    	//Answer RR number is not zero
    	if ( binary.BigEndian.Uint16(msg[6:8]) == 0){
					return
		}
    
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
		//dur := time.Duration(int(default_ttl))
		//ele.timer = time.NewTimer(dur * time.Second)

		//go start_cache_timer(ele.timer,url,req_type)

		 //log.Printf("Write cache : %s <-> %d ",  url,req_type)
		
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

