# PaoPao-Pref
## 简介
这是一个让DNS服务器预读取缓存或者压力测试的简单工具，配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用可以快速生成`redis_dns.rdb`缓存。从指定的文本读取域名列表并调用nslookup命令查询记录，docker镜像默认自带了全球前100万热门域名。   
## 警告
- 测试可能会对你的网络造成负担，请避免在网络正常使用时段进行测试。
- 若配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用，如果设置了`CNAUTO=yes`，测试前请务必设置PaoPaoDNS的docker镜像的环境变量`CNAUTO=yes`和`CNFALL=no`。
- 若配合[PaoPaoDNS](https://github.com/kkkgo/PaoPaoDNS)使用，建议PaoPao DNS的Docker的可用内存≥6 GB，否则无法缓存所有域名。
- 不建议设置过高的并发，可能会导致你的服务器崩溃或者DNS无法被有效缓存。
## 命令行参数
参数选项|值|作用
-|-|-|
-file|文件路径|指定域名列表，默认值为目录下的`domains.txt`.
-limit|并发数|指定并发数，默认值为10.
-server|DNS服务器|指定DNS服务器，默认值为空.
-line|行数|指定从第几行开始，可用于恢复进度.
-v|开关|输出域名的查询信息.
-h|开关|显示帮助信息.

## 使用二进制文件
可以从[Release](https://github.com/kkkgo/PaoPao-Pref/releases)下载对应平台编译好的二进制文件，压缩包内已经附带最新热门100万域名列表。
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
## 测试数据参考
PaoPaoDNS：4核心8G内存   
并发：10   
域名数据：100万  
运行耗时：39小时   
生成的`redis_dns.rdb`缓存文件大小：917 MB    
`used_memory_human:1.06G`

## 附录
域名数据来源： https://s3-us-west-1.amazonaws.com/umbrella-static/index.html     
PaoPao DNS Docker： https://github.com/kkkgo/PaoPaoDNS   
搭建属于自己的递归DNS：  https://blog.03k.org/post/paopaodns.html
