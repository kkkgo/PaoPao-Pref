# PaoPao-Pref
## 简介
这是一个让DNS服务器预读取缓存或者压力测试的简单工具，配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用可以快速生成`redis_dns.rdb`缓存。从指定的文本读取域名列表并调用nslookup查询记录，docker镜像默认自带了全球前100万热门域名(经过无效域名筛选)。   
## 警告
- 测试可能会对你的网络造成负担，请避免在网络正常使用时段进行测试。
- 若配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用，如果设置了`CNAUTO=yes`，测试前请务必设置PaoPaoDNS的docker镜像的环境变量`CNAUTO=yes`和`CNFALL=no`。
- 若配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用，建议PaoPao DNS的Docker的可用内存≥6 GB，否则无法缓存所有域名。
- 不建议设置过高的并发，可能会导致你的服务器崩溃或者DNS无法被有效缓存。
## 命令行参数
参数选项|值|作用
-|-|-|
-file|文件路径|指定域名列表，默认值为目录下的`domains.txt`.
-server|DNS服务器|指定DNS服务器，必需
-port|端口|指定DNS服务器端口，默认值为53.
-limit|并发数|指定并发数，默认值为10.
-timeout|5s|指定DNS查询超时时间，默认值为5s.也可以指定单位为ms.
-sleep|1500ms|指定DNS查询间隔，默认值为1500ms.值越小程序请求越快.
-line|行数|指定从第几行开始，可用于恢复进度.
-v|开关|输出域名的查询信息.
-h|开关|显示帮助信息.

## 使用二进制文件
可以从[Release](https://github.com/kkkgo/PaoPao-Pref/releases)下载对应平台编译好的二进制文件，压缩包内已经附带最新热门100万域名列表。   
你始终可以从此链接下载最新的热门100万域名列表：https://github.com/kkkgo/PaoPao-Pref/raw/main/domains.txt (经过无效域名筛选)    

## 使用Docker镜像
![pull](https://img.shields.io/docker/pulls/sliamb/paopao-pref.svg) ![size](https://img.shields.io/docker/image-size/sliamb/paopao-pref)   
![Docker Platforms](https://img.shields.io/badge/platforms-linux%2F386%20%7C%20linux%2Famd64%20%7C%20linux%2Farm%2Fv6%20%7C%20linux%2Farm%2Fv7%20%7C%20linux%2Farm64%2Fv8%20%7C%20linux%2Fppc64le%20%7C%20linux%2Friscv64%20%7C%20linux%2Fs390x-blue)   
```shell
# 帮助信息
docker run --rm -it sliamb/paopao-pref -h
# 指定DNS服务器为192.168.1.8
docker run --rm -it sliamb/paopao-pref -server 192.168.1.8
# 从第1000行开始
docker run --rm -it sliamb/paopao-pref -line 1000 -server 192.168.1.8
# 指定并发数为5
docker run --rm -it sliamb/paopao-pref -limit 5 -server 192.168.1.8
```
你也可以使用环境变量：   
环境变量名|对应选项
-|-
DNS_SERVER|-server
DNS_PORT|-port
DNS_LINE|-line
DNS_LIMIT|-limit
DNS_TIMEOUT|-timeout
DNS_SLEEP|-sleep
DNS_LOG|-v,请设置为yes/no

## 测试指标
程序的默认值兼顾性能比较低的设备，你可以适当调高/调低`limit`,`sleep`和`timeout`的值。    
`Succ rate`: 测试成功率。测试的域名在指定的timeout时间内无法解析或者解析错误（无有效A记录或者AAAA记录），会定义为失败。如果你把timeout定义的足够低，可以当缓存测试。limit和sleep的值也会影响成功率，过高的limit或者过低的sleep值也可能会导致服务器暂时无法处理.        
`Avg time`: 每个域名的查询平均处理时间。   
`Est time`: 估计的剩余时间。   

## 测试数据参考
PaoPaoDNS：4核心8G内存/`CNAUTO=yes`/`IPV6=yes`/`CNFALL=no`   
生成的`redis_dns.rdb`缓存文件大小：917 MB    
`used_memory_human:1.06G`   
该数据仅供大致参考.     
欢迎在[discussions](https://github.com/kkkgo/PaoPao-Pref/discussions)分享你的测试参数和测试数据~！

## 附录
域名数据来源(未处理)： https://s3-us-west-1.amazonaws.com/umbrella-static/index.html         
PaoPao DNS Docker： https://github.com/kkkgo/PaoPaoDNS   
搭建属于自己的递归DNS：  https://blog.03k.org/post/paopaodns.html
