package main


import (
    "database/sql"
    "fmt"
    "os"
    "flag"

    _ "github.com/mattn/go-sqlite3"
)

const RES_TABLE_NAME = "pw_resources"
const PW_TABLE_NAME  = "pw_passwords"


func main() {
    // Collect arguments 
    var cli_pw  string
    var db_name string
    flag.StringVar(&cli_pw, "key", "", "key to decrypt passwords")
    flag.StringVar(&db_name, "dbname", "store.db", "name for sqlite3 db file")
    flag.Parse()

    
    // Check for decrypt password 
    var PW string
    if len(cli_pw) > 0 {
        PW = cli_pw
    } else {
       var env_pw = os.Getenv("PWMAN_PW")
       if len(env_pw) > 0 {
           PW = env_pw
       }
    }
    if len(PW) == 0 {
        fmt.Println("No decrypt password was set, consider to use -key key or set PWMAN_PW var")
        os.Exit(1)
    }



    // Open DB conn
    db, err := sql.Open("sqlite3", db_name)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    init_tables(db)

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


func create_default_table(table string, db *sql.DB) {
    var query string
    switch {
    case table == RES_TABLE_NAME:
        query = "CREATE TABLE IF NOT EXISTS " + RES_TABLE_NAME + `(
            id INTEGER PRIMARY KEY AUTOINCREMENT, 
            resource TEXT UNIQUE
        )`
    case table == PW_TABLE_NAME:
        query = "CREATE TABLE IF NOT EXISTS " + PW_TABLE_NAME + `(
             id INTEGER PRIMARY KEY AUTOINCREMENT, 
             resource_id INTEGER, 
             username TEXT, 
             password TEXT, 
             type TEXT, 
             FOREIGN KEY(resource_id) REFERENCES pw_resources(id)
        )`
    default:
        fmt.Println("Unknown table name in create_default_table: " + table)
        return
    }

    _, err := db.Exec(query)
    if err != nil {
        panic(err)
    }
}


func init_tables(db *sql.DB) {
    // Create default tables if not exists
    if !is_table_exists(RES_TABLE_NAME, db) {
        fmt.Println("table <" + RES_TABLE_NAME + "> does not exists, creating it..")
        create_default_table(RES_TABLE_NAME, db)
    }

    if !is_table_exists(PW_TABLE_NAME, db) {
        fmt.Println("table <" + PW_TABLE_NAME + "> does not exists, creating it..")
        create_default_table(PW_TABLE_NAME, db)
    }
}
