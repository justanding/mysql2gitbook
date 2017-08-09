package main

import (
	"fmt"
	"flag"
	"log"
	"database/sql"
	"regexp"
	"os"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
)

// 数据表字段
type TableColumns struct {
	Field string
	Type string
	Null string
	Default string
	Key string
	Comment string
	Extra string
}

// 数据表
type Table struct {
	Name string
	Comment string
	Columns []TableColumns
	CreateSql string
	RealName string
	Count int //表数量 主要用于分表统计
}

var db = &sql.DB{}

var (
	emptyError = fmt.Errorf("empty")
)
var host = flag.String("h", "", "input host")
var user = flag.String("u", "", "input user")
var passwd = flag.String("p", "", "input pasword")
var dbName = flag.String("db", "", "input database name")
var filter = flag.Bool("filter", true, "是否去重")
var baseName = "./data/" //数据存放目录

func init(){
	var err error
	flag.Parse()
	dbstr := *user + ":" + *passwd + "@tcp(" + *host + ")/" + *dbName
	baseName += *dbName + "/"

	os.RemoveAll(baseName)
	fmt.Println(baseName)
	if err := os.MkdirAll(baseName, 0777); err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("mysql", dbstr)
	if err != nil {
		fmt.Println("dbis:", dbstr)
		log.Fatal(err)
	}
}

func main() {

	tables, err := showTables()
	if err != nil {
		log.Fatal(err)
	}

	ts, err := filterDuplicate(tables)

	for k, v := range ts {
		err := v.showTableStatus()
		if err != nil {
			log.Fatal(err)
		}
		
		err = v.showColumns()
		if err != nil {
			log.Fatal(err)
		}

		err = v.showCreateTable()
		if err != nil {
			log.Fatal(err)
		}

		ts[k] = v
	}

	createGitbook(ts)
}

func showTables() ([]string, error) {
	var tables []string

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	for rows.Next() {
		var Tables string
		err := rows.Scan(&Tables)
		if err != nil {
			return tables, err
		}
		tables = append(tables, Tables)
	}

	return tables, nil
}

func (t *Table)showTableStatus() error {
	rows, err := db.Query("SHOW TABLE status WHERE Name='" + t.RealName + "'")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var Name, Comment string
		var Engine, Row_format, Create_time, Update_time, Check_time,Collation,Create_options interface{}
		var Version, Rows, Avg_row_length, Data_length, Max_data_length, Index_length,Data_free,Auto_increment,Checksum interface{}
		err := rows.Scan(
			&Name, &Engine, &Version, &Row_format, &Rows,
			&Avg_row_length, &Data_length, &Max_data_length,
			&Index_length, &Data_free, &Auto_increment,
			&Create_time, &Update_time, &Check_time,
			&Collation, &Checksum, &Create_options, &Comment,
		)
		if err != nil {
			return err
		}
		t.Comment = Comment
		return nil
	}

	return emptyError
}

func (t *Table) showColumns() error {
	var columns []TableColumns

	rows, err := db.Query("show full columns from " + t.RealName)
	if err != nil {
		 return err
	}
	defer rows.Close()

	for rows.Next(){
		var Field, Type, Null, Key, Extra, Privileges, Comment string
		var Default, Collation interface{}
		err := rows.Scan(
			&Field, &Type, &Collation,
			&Null, &Key, &Default,
			&Extra, &Privileges, &Comment,
		)
		if err != nil {
			return  err
		}

		d := ""
		switch value := Default.(type) {
		case int:
			d = strconv.Itoa(value)
		case string:
			d = value
		default:
			d = ""
		}

		column := &TableColumns{
			Field:Field,
			Type:Type,
			Default:d,
			Key:Key,
			Null:Null,
			Comment:Comment,
			Extra:Extra,
		}
		columns = append(columns, *column)
	}

	t.Columns = columns
	return nil
}

func (t *Table) showCreateTable() error {

	rows, err := db.Query("show create table " + t.RealName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var Name, CreateSql string

		err := rows.Scan(&Name, &CreateSql)
		if err != nil {
			 return err
		}
		t.CreateSql = CreateSql
		return nil
	}

	return emptyError
}

func filterDuplicate(tables []string) (map[string]Table, error){
	var Tables = map[string]Table{}

	re := regexp.MustCompile("_?\\d+$")
	for _, v := range tables {
		src := v
		if *filter == true {
			src = re.ReplaceAllString(v, "")
		}

		t, ok := Tables[src]
		if ok == false {
			t = Table{
				Name : src,
				RealName : v,
				Count: 1,
			}

			Tables[src] = t
		} else {
			t.Count += 1
			Tables[src] = t
		}
	}

	return Tables, nil
}

func createGitbook(tables map[string]Table) {

	readme := "### 目录 \n\n"
	summary := "* [目录](README.md)\n"
	for _, v := range tables {
		filename := v.Name + ".md"

		linkName := v.Name
		if len(v.Comment) > 0 {
			linkName = v.Comment
		}

		list := "* [" + linkName + "](" + filename + ")\n"
		readme += list
		summary += "    " + list

		s := "## " + v.Name + "\n"
		if len(v.Comment) == 0 {
			v.Comment = "请添加注释"
		}
		s += "	" + v.Comment + "\n"
		s += "	 共" + strconv.Itoa(v.Count) + "张表\n\n"

		s += "### 表结构说明 \n\n"

		s += "|Field|Type|Key|Default|Null|Comment|Extra\n"
		s += "|-----|----|---|-------|----|---|-----\n"
		for _, c := range v.Columns {
			s += "| " +
				c.Field + " | " +
				c.Type + " | " +
				c.Key + " | " +
				c.Default + " | " +
				c.Null + " | " +
				c.Comment + " | " +
				c.Extra + "\n"
		}
		s += "\n"

		s += "### sql语句 \n\n"
		s += "```sql\n"
		s += v.CreateSql + "\n"
		s += "```"

		writeFile(filename, s)

	}

	writeFile("SUMMARY.md", summary)
	writeFile("README.md", readme)
}

func writeFile(filename, content string) error {
	realfn := baseName + filename

	f, err := os.Create(realfn)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err =f.WriteString(content)
	return err
}

