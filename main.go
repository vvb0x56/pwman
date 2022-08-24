package main


import (
    "database/sql"
    "fmt"
    "os"
    "flag"
    "strconv"

    _ "github.com/mattn/go-sqlite3"
)

const RES_TABLE_NAME = "pw_resources"
const PW_TABLE_NAME  = "pw_passwords"


type PasswordRecord struct {
    resource string
    user     string
    passwd   string
    app      string
}


func main() {
    // Collect arguments 
    var cli_pw       string
    var db_name      string
    var ls       bool
    var add      bool
    var creat_tables bool

    flag.StringVar(&cli_pw,  "key",  "",         "key to decrypt passwords")
    flag.StringVar(&db_name, "db",   "store.db", "name for sqlite3 db file")
    flag.BoolVar(&creat_tables, "create-tables", false,      
                 "create tables (you need to drop it if exists)")
    //flag.IntVar(&ls_pw,     "lspw", false,      "list available passwords")

    // LS 
    flag.BoolVar(&ls,    "ls",   false,      "list available resources")
    // ADD 
    flag.BoolVar(&add, "a",  false, "add new password: resource name/id username password type(web)")
    flag.Parse()


    // Open DB conn
    db, err := sql.Open("sqlite3", db_name)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Create tables can be done without password, so let's handle it first
    if creat_tables {
        check_tables(db_name, db)
        return 
    }

    // ls all available resoureces without passwords
    if ls {
        print_resources(db)
        return 
    }
    
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

    // ADD 
    if add {
        if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
            flag.Usage()
            return
        }

        var record PasswordRecord

        record.resource = flag.Arg(0)
        record.user     = flag.Arg(1)
        record.passwd   = flag.Arg(2)
        record.app      = "web" // default

        if len(flag.Args()) == 4 {
            record.app = flag.Arg(3)
        }

        add_password(&record, db)
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


func check_tables(db_name string, db *sql.DB) {
    // Create default tables if not exists
    if !is_table_exists(RES_TABLE_NAME, db) {
        fmt.Println("table <" + RES_TABLE_NAME + "> does not exists in [" + db_name + "], creating it..")
        create_default_table(RES_TABLE_NAME, db)
    } else {
        fmt.Println("table <" + RES_TABLE_NAME + "> already exists in [" + db_name + "], do nothing.")
    }

    if !is_table_exists(PW_TABLE_NAME, db) {
        fmt.Println("table <" + PW_TABLE_NAME + "> does not exists in [" + db_name + "], creating it..")
        create_default_table(PW_TABLE_NAME, db)
    } else {
        fmt.Println("table <" + PW_TABLE_NAME + "> already exists in [" + db_name + "], do nothing.")
    }
}


func print_resources(db *sql.DB) {
    query := "SELECT id, resource FROM " + RES_TABLE_NAME 
    rows, err := db.Query(query)
    if err != nil {
        panic(err)
    }
    defer rows.Close()

    for rows.Next() {
        var (
            id       int
            resource string
        )
        if err := rows.Scan(&id, &resource); err != nil {
            panic(err)
        }
        fmt.Printf("%d: %s\n", id, resource)
    }
}


func add_password(record *PasswordRecord, db *sql.DB) {
    var id    int 
    var query string

    if _, err := strconv.Atoi(record.resource); err == nil {
        query = "SELECT id FROM " + RES_TABLE_NAME + " WHERE id = ?"
    } else {
        query = "SELECT id FROM " + RES_TABLE_NAME + " WHERE resource = ?"
    }

    // find id for resource or create it 
    if err := db.QueryRow(query, record.resource).Scan(&id); err != nil {
        if err == sql.ErrNoRows {
            fmt.Println("no resource found, creating it..")
            id, err = insert_resource(record.resource, db)
            if err != nil {
                panic(err)
            }
        } else {
            panic(err)
        }
    }

    // create password for resource id 

    fmt.Println("res name is id", id)
    _, err := insert_password(id, record, db)
    if err != nil {
        panic(err)
    }
}


func insert_resource(resource string, db *sql.DB) (id int, err error) {
    err = db.QueryRow("INSERT INTO " + RES_TABLE_NAME + "(resource) VALUES(?) RETURNING id", 
                       resource).Scan(&id)
    return id, err
}


func insert_password(id int, record *PasswordRecord, db *sql.DB) (res sql.Result, err error) {
    res, err = db.Exec("INSERT INTO " + PW_TABLE_NAME + `(resource_id, username, password, type) 
                        VALUES($1, $2, $3, $4)`, 
                        id, 
                        record.user, 
                        record.passwd, 
                        record.app)
    return res, err
}
