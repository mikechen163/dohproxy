# dohproxy

DNS over HTTPS proxy written in golang.

dohproxy is a local dns proxy program. It can run on the local computer, or on the router.

The main features implemented:

1 dns classification forwarding. According to the domain name of dns, dns requests are forwarded to different dns servers.

  For any dns request, first query the cn.txt file, if there is this domain name in the file, then forward the request to the domestic dns server. The default is 223.5.5.5 129.29.29.29. You can use the -innserver parameter to change the default domestic dns server.

  If the domain name is not found in the cn.txt file, it means it is an overseas domain. Then the domain name will be forwarded directly to the configured overseas server. The default is https://8.8.8.8/dns-query.
Use the -dohserver parameter to modify the overseas dns server.
  
2 The overseas server supports udp, tcp, and doh protocols in the following format:
. /dohproxy -dohserver 8.8.8.8,8.8.4.4
. /dohproxy -dohserver tcp://8.8.8.8:53,tcp://8.8.4.4:53
. /dohproxy -dohserver https://8.8.8.8/dns-query 

For any dns request, first check cn.txt, if found, then forward the dns request to the domestic server, otherwise forward the dns request to an overseas dns server such as dns.google.

3 Support edns subnet feature. You can specify the subnet parameter when forwarding to overseas dns servers. The specific format is :

. /dohproxy -subnet 100.22.33.0 -dohserver 8.8.8.8

4 Support overseas dns entry caching, activated by default. The maximum length of the cache is ttl. Modify it with the -ttl parameter. 

5 If you use tcp protocol to access overseas dns servers, then you can enable tcp reuse mechanism, which is activated by default. This allows multiple dns requests to use the same tcp link, saving the number of tcp links and increasing the speed of dns queries.

dohproxy是一个本地dns代理程序. 可以运行在本地计算机上,或者运行在路由器上.

实现的主要特性:

1 dns分类转发. 根据dns的域名归属,转发dns请求到不同的dns服务器.

对于任意的dns请求,首先查询 cn.txt文件, 如果文件里面有这个域名,就把请求转发给国内的dns服务器. 缺省223.5.5.5 129.29.29.29. 可以使用 -innserver 参数,修改缺省的国内dns服务器.

如果cn.txt文件中找不到这个域名,说明属于海外域名.则直接把域名转发给配置的海外服务器. 缺省情况是 https://8.8.8.8/dns-query.
使用 -dohserver 参数,修改海外的dns服务器.
  
2 海外服务器支持 udp,tcp,doh协议,具体格式为:
./dohproxy  -dohserver 8.8.8.8,8.8.4.4
./dohproxy  -dohserver tcp://8.8.8.8:53,tcp://8.8.4.4:53
./dohproxy  -dohserver https://8.8.8.8/dns-query 

对于任何 dns 请求，首先检查 cn.txt ，如果找到，则将 dns 请求转发到国内服务器，否则将 dns 请求转发到 dns.google 等海外 dns 服务器。

3 支持edns subnet特性.可以指定转发给海外dns服务器时候的subnet参数. 具体格式为:

./dohproxy -subnet 100.22.33.0 -dohserver 8.8.8.8

4 支持海外dns条目缓存,缺省激活. 缓存的最大时长为ttl. 通过-ttl参数修改. 

5 如果使用tcp协议访问海外dns服务器,那么可以启用tcp 重用机制,缺省激活. 这样多个dns请求可以使用同一个tcp链路,节约tcp链接数目,提高dns查询速度.



# Install
  
    
   download [release](https://github.com/mikechen163/dohproxy/releases), extract to local directory.
   
    ./dohproxy -h //for help information
   
   The following command starts dohproxy in the background, listens to port 53, and sets the edns subnet to 100.22.33.0
   
   下面这条命令,启动dohproxy在后台,监听53号端口,设定edns subnet子网为 100.22.33.0
   
    sudo nohup ./dohproxy -port 53 -subnet 100.22.33.0 -dohserver https://8.8.8.8/dns-query > /dev/null  2>&1 &

   
   
    
# Usage

  Usage of dohproxy: dohproxy -h
  
    Usage of ./dohproxy:
    -block string
    	default ad keyword list file (default "block.txt")
    -cached
    	set cache mode : experiment! (default true)
    -chn string
    	default domestic domain list file (default "cn.txt")
    -debug
    	print debug logs
    -dohserver string
    	DNS Over HTTPS server address (default "https://8.8.8.8/dns-query")
    -fallback
    	set fallback mode
    -host string
    	interface to listen on (default "localhost")
    -innserver string
    	Domestic Dns server address (default "223.5.5.5,119.29.29.29")
    -port int
    	dns port to listen on (default 5353)
    -subnet string
    	set edns default subnet (default "0.0.0.0")
    -tcp_reuse
    	dns over tcp and tcp_reuse_flag : experiment! (default true)
    -ttl int
    	default oversea ttl length (default 7200)      
 

 
# LICENSE
Copyright (C) 2020 mike chen

This program is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program. If not, see http://www.gnu.org/licenses/.
  


