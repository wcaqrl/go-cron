* 编译master
```shell script
cd master/main/
go build -o ../bin/master
```
* 创建 master 配置文件软链接
```shell script
cd master/bin/
ln -s master-prod.json master.json
```
* 启动master
```shell script
cd master/bin
./master 
或
./master -config ./master.json

* 浏览器打开管理后台
```
master 启动之后, 可访问该机 8070 端口即可进入管理后台,如： http://172.17.12.100:8070
开发环境和生产环境的用户名和密码请在 master.json 配置文件中查找
```

* 编译worker
```shell script
cd worker/main/
go build -o ../bin/worker
```

* 创建 worker 配置文件软链接
```shell script
cd worker/bin/
ln -s worker-prod.json worker.json
```
* 启动worker
```shell script
cd worker/bin
./worker
或
./worker -config ./worker.json
```

* 定时任务日志索引
```
索引名称
fun-job-log
```
* Nginx访问日志索引
```
索引名称
orange_log | open-api_log | tvapi_log | tvcms_log | ...
```
* 配置文件master|worker.json参数说明:
```
详见项目具体配置文件
``` 

* 脚本执行顺序:
```
Step 1 
启动 master
Step 2
启动各个节点的 worker
``` 
