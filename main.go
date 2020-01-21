package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"bytes"
)

func main() {
	host := flag.String("host", "localhost", "interface to listen on")
	port := flag.Int("port", 5353, "dns port to listen on")
	dohserver := flag.String("dohserver", "https://mozilla.cloudflare-dns.com/dns-query", "DNS Over HTTPS server address")
	debug := flag.Bool("debug", false, "print debug logs")
	flag.Parse()

	if *debug {
		log.SetFlags(log.Lshortfile)
	} else {
		log.SetFlags(0)
	}

	if err := newUDPServer(*host, *port, *dohserver); err != nil {
		log.Fatalf("could not listen on %s:%d: %s", *host, *port, err)
	}
}

func getUrl(localBuf []byte) string {

 
	len2 := len(localBuf)
	if len2 == 0 {
		return ""
	}


	var s bytes.Buffer
	
	i := 0
	for {

		

		if  (i > (len2 -1)) {
			return s.String()
		}

		c := localBuf[i]


		if  (c == 0) {
			return s.String()
		}

		printable := false
		if (c >= 'a') && (c <= 'z') {
			printable = true
		}
		if (c >= 'A') && (c <= 'Z') {
			printable = true
		}
		if (c >= '1') && (c <= '9') {
			printable = true
		}

		if (c == '-') || (c == '_') || (c == '.') {
			printable = true
		}

		tc := string(c)
		if printable == false {
			tc = "."
			//if c != "." { //other char is invalid
				//return s.String()
			//}
		}

		//s = s + tc
		s.WriteString(tc)
		i = i + 1

	}

	return s.String()

}

func newUDPServer(host string, port int, dohserver string) error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		return err
	}
	for {
		var raw [512]byte
		n, addr, err := conn.ReadFromUDP(raw[:512])
		if err != nil {
			log.Printf("could not read: %s", err)
			continue
		}
		//log.Printf("new connection from %s:%d", addr.IP.String(), addr.Port)
		log.Printf("query : %s ", getUrl(raw[13:]))
		go proxy(dohserver, conn, addr, raw[:n])
	}
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
