# mysql2gitbook
将表结构导出到gitbook；数据存放在命令执行目录的data目录

有简单的去重分表的处理，将处理table_name(_?\d+)$成table_name

example:
    go run main.go -h 127.0.0.1:3306 -u root -p root -db dbname -filter true
