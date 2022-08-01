# dohproxy

DNS over HTTPS proxy written in golang.

This program is inspired by https://github.com/satran/dohproxy. I forked the original code, add support for udp server as upstream servers  and ad block function.

Then you got a dns proxy, which listen on 5353(default), and forward the request to upstream dns servers for normal domestic sites, and using dns over https for blocked sites like google facebook etc.

简单的 DNS 转发器

对于任何 dns 请求，首先检查 cn.txt ，如果找到，则将 dns 请求转发到国内服务器，否则将 dns 请求转发到 dns.google 等海外 dns 服务器。

用法：

对海外 dns req 使用 dns over http 协议。
./dohproxy -dohserver https://8.8.8.8/dns-query

或者
使用 udp 处理海外 dns 请求
./dohproxy -dohserver 8.8.8.8:53

./dohproxy -h打印帮助信息


# install
  
    
    git clone --depth 1 https://github.com/mikechen163/dohproxy.git
    cd mikechen163/dohproxy
    go build
    
# usage

  Usage of dohproxy: dohproxy -h
        
    -block string
    	default ad keyword list file (default "block.txt")
    -chn string
    	default domestic domain list file (default "cn.txt")
    -debug
    	print debug logs
    -dohserver string
    	DNS Over HTTPS server address (default "https://8.8.8.8/dns-query")
    -host string
    	interface to listen on (default "localhost")
    -innserver string
    	Domestic Dns server address (default "223.5.5.5,119.29.29.29")
    -port int
    	dns port to listen on (default 5353)
    -ttl int
    	default oversea ttl length (default 3600)
    	
   
   The following command starts the proxy,listen on port 53, check url against domestic domain list file(cn.txt), if success, then forward the dns request to upstream dns server using udp protocol. if fail,forward the request to dohserver using dns over http protocol.
    	
    sudo nohup ./dohproxy -port 53  >> run.log 2>&1 &
    
    
# 使用说明

  cn.txt包含了所有的中文域名,如果请求的域名在这个文件里面,就直接访问国内的dns服务器.如果不在这个文件里面,就通过doh协议,访问正确的doh服务器.
  block.txt包含了广告的域名,直接丢弃.   

 
# LICENSE
Copyright (C) 2020 mike chen

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see http://www.gnu.org/licenses/.
  


