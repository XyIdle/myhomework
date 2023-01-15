# 测试过程和输出

1. client循环访问

```bash
Shell> while
do
    curl http://127.0.0.1:8080
    sleep 1
done
hello
hello
hello
hello
hello
服务已关闭curl: (7) Failed to connect to 127.0.0.1 port 8080 after 10 ms: Connection refused
curl: (7) Failed to connect to 127.0.0.1 port 8080 after 10 ms: Connection refused
curl: (7) Failed to connect to 127.0.0.1 port 8080 after 11 ms: Connection refused
curl: (7) Failed to connect to 127.0.0.1 port 8080 after 11 ms: Connection refused
curl: (7) Failed to connect to 127.0.0.1 port 8080 after 11 ms: Connection refused
```


2. server启动和停止

```bash
./homework_graceful_shutdown
^C2023/01/16 00:55:01 开始关闭应用，停止接收新请求
2023/01/16 00:55:01 等待正在执行请求完结
2023/01/16 00:55:01 开始关闭服务器
2023/01/16 00:55:01 服务器admin关闭中
2023/01/16 00:55:01 服务器business关闭中
2023/01/16 00:55:02 服务器business已关闭
2023/01/16 00:55:02 服务器admin已关闭
2023/01/16 00:55:02 开始执行自定义回调
2023/01/16 00:55:02 刷新缓存中……
2023/01/16 00:55:04 缓存被刷新到了 DB
2023/01/16 00:55:04 开始释放资源
2023/01/16 00:55:05 应用关闭
```
