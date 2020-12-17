# 简介

mysql -> mariadb columnStore的同步工具，支持全量同步和增量同步。
mariadb是mysql的社区开源版，columnStore则是mariadb的插件式的列式存储引擎。

MariaDB ColumnStore专为大数据扩展而设计，可处理PB级数据，线性可伸缩性和出色的性能，并能对分析查询进行实时响应。 它利用列式存储，压缩，即时投影以及水平和垂直分区的I / O优势在分析大型数据集时提供了出色的性能。（这是官方的介绍，[https://mariadb.com/kb/en/mariadb-columnstore/](https://mariadb.com/kb/en/mariadb-columnstore/)）

所以此项目用于将mysql数据同步到mariadb columnStore中，以实现复杂的sql查询和数据分析，可以把mariadb columnStore当成数据仓库的解决方案。


# 实现原理

全量同步：
导出mysql库表结构，经过sql兼容转换，在mariadb columnStore中建立相应的库表；
mysql 通过 `SELECT INTO OUTFILE` 高速导出数据，mariadb columnStore通过 `LOAD DATA LOCAL INFILE` 高速导入数据（内部会转换成cpimport方式导入）。

增量同步：
模拟mysql slave, 监听binlog日志，binlog event解析成相应的sql然后在mariadb columnStore中消费。
类似于阿里巴巴 [canal](https://github.com/alibaba/canal) 的实现。增量同步支持断点重新订阅消费。那么我是如何保证binlog重新消费幂等性的？对于DDL操作，如果重复执行，mysql会报错返回，如create table、alter table 等操作，所以不会造成数据的一致性问题。DML操作：insert语句，消费binlog之前我会先判断目标数据库是否存在相同主键的数据，如果有，则跳过，这就得保证mysql的表都得有主键； update语句，采用乐观锁机制，binlog解析成sql之前，把更改前的信息作为where条件组装成新的update语句，如果重复消费，那么将会update失败；delete语句和update语句类似地采用乐观锁机制避免重复消费造成数据一致性问题。总的来说，只要保证binlog到达消费端之前是有序的，那么就能保证消费的幂等性。注：尚未解决的问题是由于colomnStore引擎不支持事务，如果mysql执行的某个事务中包含多个dml语句，那么在消费binlog同步数据的时候可能在某一时刻会出现数据不一致的问题，因为binlog恢复的sql在消费端是一条一条执行的，不能像mysql的Innodb引擎一样，执行一组事务操作。


# 如何使用

```
# 全量同步
./mcs_etl -model full

# 增量同步
./mcs_etl -model increment
```


## 下面完整演示一下如何使用该工具

下载该项目

```
git clone https://github.com/yutianyong125/mcs_etl.git
cd mcs_etl
```

启动mysql，方便起见，这里使用docker，仅测试，不考虑数据挂载问题，这里我把secure-file-priv配置的目录挂载出来，是为了导入数据的时候用到，另需要指定使用自定义的mysql配置文件

```
vim /tmp/conf.d/my.cnf

# 保存为以下配置

[mysqld]
log_bin = mysql_bin # 开启binlog日志并指定binlog日志命名前缀
binlog_format=ROW # 指定binlog格式
server_id = 1 # 指定master server_id
secure_file_priv=/tmp/mysql-files # SELECT INTO OUTFILE 语句需要开启这个配置
```

```
mkdir /tmp/mysql-files # 先创建好存放导出数据的文件夹

docker run --rm -d --name mysql -v /tmp/conf.d:/etc/mysql/conf.d -v /tmp/mysql-files:/tmp/mysql-files -e MYSQL_ROOT_PASSWORD=123456 -p 3306:3306 mysql:5.7 --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci --secure-file-priv=/tmp/mysql-files
```

使用存储过程添加测试数据

```
-- 创建test库
create database test;
use test;

-- 创建user表

CREATE TABLE user(
  id INT NOT NULL AUTO_INCREMENT,
  uname VARCHAR(20) NOT NULL,
  sex VARCHAR(5) NOT NULL,
  score INT NOT NULL,
  copy_id INT NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=INNODB CHARSET=utf8;

-- 存储过程插入10万条数据

DROP PROCEDURE IF EXISTS add_user;  
DELIMITER //
    create PROCEDURE add_user(in num INT)
    BEGIN
        DECLARE rowid INT DEFAULT 0;
        DECLARE firstname CHAR(1);
        DECLARE name1 CHAR(1);
        DECLARE name2 CHAR(1);
        DECLARE sex CHAR(1);
        DECLARE score CHAR(2);
        DECLARE uname CHAR(3);
        WHILE rowid < num DO
        SET firstname = SUBSTRING('赵钱孙李周吴郑王林杨柳刘孙陈江阮侯邹高彭徐',FLOOR(1+21*RAND()),1); 
        SET name1 = SUBSTRING('一二三四五六七八九十甲乙丙丁静景京晶名明铭敏闵民军君俊骏天田甜兲恬益依成城诚立莉力黎励',ROUND(1+43*RAND()),1); 
        SET name2 = SUBSTRING('一二三四五六七八九十甲乙丙丁静景京晶名明铭敏闵民军君俊骏天田甜兲恬益依成城诚立莉力黎励',ROUND(1+43*RAND()),1); 
        SET sex=FLOOR(0 + (RAND() * 2));
        SET score= FLOOR(40 + (RAND() *60));
        SET rowid = rowid + 1;
        SET uname = CONCAT(firstname,name1,name2);
        insert INTO user (uname,sex,score,copy_id) VALUES (uname,sex,score,rowid);  
        END WHILE;
    END //
DELIMITER ;

call add_user(100000);

-- 创建test1库

create database test1;
use test1;

-- 创建表t1

CREATE TABLE `t` (
  `id` int(11) NOT NULL,
  `a` int(11) DEFAULT NULL,
  `b` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `a` (`a`),
  KEY `b` (`b`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- 存储过程插入数据

delimiter //
create procedure idata()
begin
  declare i int;
  set i=1;
  while(i<=100000)do
    insert into t values(i, i, i);
    set i=i+1;
  end while;
end //
delimiter ;
call idata();
```

启动mariadb columnStore

```
docker run --rm -d --name mcs -eMARIADB_ROOT_PASSWORD=123456 -p 3307:3306 mariadb/columnstore:1.2.3
```

修改项目配置文件, conf/etl.toml

```
# 增量ETL配置段
[IncrementEtl]
  StartFile = "" # 开始同步的binlog文件
  StartPosition = 0 # 开始同步的binlog位置
  ServerId = 100 # 同步slave的serverId，只要不与master serverId相同即可

# 全量ETL配置段
[FullEtl]
  # mysql数据导出存放文件夹
  OutFileDir = "/tmp/mysql-files/"

# 全量ETL规则，Schema: 数据库， Tables: table数组
# eg: 导出整个库的表 tables=["*"]，导出特定的表 tables=["test1", "test2"]
[[Rule]]
  Schema = "test"
  Tables = ["user"]
[[Rule]]
  Schema = "test1"
  Tables = ["*"]

# mysql 数据库配置
[Source]
  Host = "127.0.0.1"
  Port = 3306
  User = "root"
  Pwd = "123456"

# mariadb columnStore 数据库配置
[Target]
  Host = "127.0.0.1"
  Port = 3307
  User = "root"
  Pwd = "123456"
```

执行二进制文件，指定全量同步模式

```
./mcs_etl -model full
```

成功输出如下，因为是多协程异步执行任务的，所以总耗时可能与etl时长最大的表相差无几

```
同步`test1`.`t` 耗时 2.025725559s
同步`test`.`user` 耗时 4.358782152s
fullEtl 耗时 4.373473788s
```

增量方式进行同步步骤如下：

mysql运行 show master status 查看当前binlog日志文件和Position，修改 conf/etl.toml 配置文件中的IncrementEtl配置段

```
# 增量ETL配置段
[IncrementEtl]
  StartFile = "mysql_bin.000004" # 开始同步的binlog文件
  StartPosition = 154 # 开始同步的binlog位置
  ServerId = 100 # 同步slave的serverId，只要不与master serverId相同即可
```

执行二进制文件，指定增量同步模式，此时会进入循环监听等待消费状态

```
./mcs_etl -model increment
```

接下来在mysql做一些增删改操作

```
update `test`.`user` set sex = 0 where id = 1;

insert into `test1`.`t` values (100001, 100001, 100001);

delete from `test1`.`t` where id = 100001;
```

控制台输出日志如下

```
binlog恢复的语句：
update `test`.`user` set `id` = '1',`uname` = '邹二君',`sex` = '0',`score` = '59',`copy_id` = '1' where `id` = '1' and `uname` = '邹二君' and `sex` = '1' and `score` = '59' and `copy_id` = '1'
==>
兼容处理转换后的语句：
update `test`.`user` set `id` = '1',`uname` = '邹二君',`sex` = '0',`score` = '59',`copy_id` = '1' where `id` = '1' and `uname` = '邹二君' and `sex` = '1' and `score` = '59' and `copy_id` = '1'

执行结果：
执行成功

binlog恢复的语句：
insert into `test1`.`t` (`id`,`a`,`b`) values ('100001','100001','100001')
==>
兼容处理转换后的语句：
insert into `test1`.`t` (`id`,`a`,`b`) values ('100001','100001','100001')

执行结果：
执行成功

binlog恢复的语句：
delete from `test1`.`t` where `id` = '100001' and `a` = '100001' and `b` = '100001'
==>
兼容处理转换后的语句：
delete from `test1`.`t` where `id` = '100001' and `a` = '100001' and `b` = '100001'

执行结果：
执行成功
```
