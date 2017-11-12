一、服务器搭建流程
1. 安装并启动mysql服务器
	wget http://repo.mysql.com/mysql-community-release-el7-5.noarch.rpm
	sudo rpm -ivh mysql-community-release-el7-5.noarch.rpm
	yum update
	sudo yum install mysql-server
	sudo systemctl start mysqld
2. 导入Mysql表结构
3. 安装并启动nsqd消息队列
	修改本机名字为 127.0.0.1 ecs-6e63.novalocal
	wget https://s3.amazonaws.com/bitly-downloads/nsq/nsq-1.0.0-compat.linux-amd64.go1.8.tar.gz
	nohup ./nsqlookupd >nsqlookupd.log 2>&1 &
	nohup ./nsqd --lookupd-tcp-address=127.0.0.1:4160 >nsqd.log 2>&1 &	
	nohup ./nsqadmin --lookupd-http-address=127.0.0.1:4161 >nsqadmin.log 2>&1 &
4. 编译并启动tcpBoltDB
	git@github.com:LovelyLich/translatechat.git
	cd translatechat/tcpBoltDB/; go build
	./tcpBoltDB
5. 编译并启动nsq_translate
	cd translatechat/nsq_translate; go build	
  	./nsq_translate

