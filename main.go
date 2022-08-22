package main


import (
    "database/sql"
    "fmt"
    "os"
    "log"

    _ "github.com/mattn/go-sqlite3"
)


const RES_TABLE_NAME = "pw_resources"
const PW_TABLE_NAME  = "pw_passwords"


func main() {
    create_tables()
    // Check for password in env
    var PW = os.Getenv("PWMAN_PW")
    if len(PW) == 0 {
        log.Fatalln("No PWMAN_PW env variable was set.") 
    }

    db, err := sql.Open("sqlite3", "store.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    if !is_table_exists(RES_TABLE_NAME, db) {
        fmt.Println("table <" + RES_TABLE_NAME + "> does not exists")
        return 
    }

    if !is_table_exists(PW_TABLE_NAME, db) {
        fmt.Println("table <" + PW_TABLE_NAME + "> does not exists")
        return 
    }



}


func is_table_exists(tablename string, db *sql.DB) bool {
    var count int
    // Right way to handle QueryRow
    // https://www.calhoun.io/querying-for-a-single-record-using-gos-database-sql-package/
    row := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tablename) 
    if err := row.Scan(&count); err != nil {
        panic(err)
    }
    if count == 1 {
        return true
    }
    return false
}


//func create_tables(db *sql.DB) {
func create_tables() {
    res_query := "create table " + RES_TABLE_NAME + "(id INTEGER PRIMARY KEY, resource TEXT)"
    pw_query  := "create table " + PW_TABLE_NAME + `(
        id INTEGER PRIMARY KEY, 
         resource_id INTEGER, 
         username TEXT, 
         password TEXT, 
         type TEXT, 
         FOREIGN KEY(resource_id) REFERENCES pw_resources(id)
    )`

    fmt.Println(res_query)
    fmt.Println(pw_query)

    return 
}


//func create_table(query string, db *sql.DB) {
//    return 
//}

