package main


import (
    "database/sql"
    "fmt"
    "os"
    "io"
    "flag"
    "strconv"
    "crypto/aes"
    "crypto/cipher"
    "crypto/sha1"
    "crypto/rand"
    "encoding/hex"

    "golang.org/x/crypto/pbkdf2"

    _ "github.com/mattn/go-sqlite3"

)

const RES_TABLE_NAME = "pw_resources"
const PW_TABLE_NAME  = "pw_passwords"
const SALT           = "C2X31234H991K331"
const KEY_LEN        = 32  // to use AES-256

var PW_KEY []byte

type PasswordRecord struct {
    resource string
    user     string
    passwd   string
    app      string
}


func main() {
    // Collect arguments 
    var cli_pw, db_name         string
    var ls, lspw, add, rm, rmpw bool
    var creat_tables            bool

    flag.StringVar(&cli_pw,  "key",  "",         "key to decrypt passwords")
    flag.StringVar(&db_name, "db",   "store.db", "name for sqlite3 db file")
    flag.BoolVar(&creat_tables, "create-tables", false,      
                 "create tables (you need to drop it if exists)")
    //flag.IntVar(&ls_pw,     "lspw", false,      "list available passwords")

    // LS LSPW
    flag.BoolVar(&ls,   "ls",  false, "list available resources")
    flag.BoolVar(&lspw, "lspw",false, "list passwords")
    // ADD 
    flag.BoolVar(&add, "a",    false, "add new password: resource name/id username password type(web)")
    flag.BoolVar(&rm,  "rm",   false, "remove resource")
    flag.BoolVar(&rmpw,"rmpw", false, "remove password")
    flag.Parse()


    // Open DB conn
    db, err := sql.Open("sqlite3", db_name)
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Create tables can be done without password, so let's handle it first
    if creat_tables {
        checkTables(db_name, db)
        return 
    }

    // ls all available resoureces without passwords
    if ls {
        printRes(db)
        return 
    }
    
    // Is encryption password set?
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

    // make PW len eq to 32, as required for aes
    PW_KEY = pbkdf2.Key([]byte(PW), []byte(SALT), 4096, KEY_LEN, sha1.New)

    // LSPW
    if lspw {
        if len(flag.Args()) == 0 {
            showPW(db, "")
            return
        }
        showPW(db, flag.Arg(0))
        return
    }

    // ADD 
    if add {
        if len(flag.Args()) < 3 || len(flag.Args()) > 4 {
            fmt.Println("..Usage: ./pwman -a RES_ID USER PASSWD APP")
            return
        }

        var record PasswordRecord

        record.resource = flag.Arg(0)
        record.user     = flag.Arg(1)
        record.passwd   = encrypt(flag.Arg(2), PW_KEY)
        record.app      = "web" // default

        if len(flag.Args()) == 4 {
            record.app = flag.Arg(3)
        }

        addPW(&record, db)
        return 
    }

    // RM 
    if rm {
        if len(flag.Args()) != 1 {
            fmt.Println("..Need an id of the removing resource") 
            return
        }
        if _, err := strconv.Atoi(flag.Arg(0)); err != nil {
            fmt.Println("..Need an numeric id.") 
            return
        }
        delResource(flag.Arg(0), db)
        return 
    }
    // RMPW
    if rmpw {
        if len(flag.Args()) != 1 {
            fmt.Println("..Need an id of the removing password") 
            return
        }
        if _, err := strconv.Atoi(flag.Arg(0)); err != nil {
            fmt.Println("..Need an numeric id of password.") 
            return
        }
        delPassword(flag.Arg(0), db)
        return 
    }
}


func encrypt(stringToEncrypt string, key []byte)(encryptedString string) {
    plaintext := []byte(stringToEncrypt)

    block, err := aes.NewCipher(key)
    if err != nil {
        panic(err.Error())
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        panic(err.Error())
    }

    nonce := make([]byte, aesGCM.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        panic(err.Error())
    }

    ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

    return hex.EncodeToString(ciphertext)
}

func decrypt(encryptedString string, key []byte) (decryptedString string) {
    enc, _ := hex.DecodeString(encryptedString)

    block, err := aes.NewCipher(key)
    if err != nil {
        panic(err.Error())
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        panic(err.Error())
    }

    nonceSize := aesGCM.NonceSize()

    nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

    plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return err.Error()
    }
    return string(plaintext)
}

func isTablePresents(tablename string, db *sql.DB) bool {
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


func createTables(table string, db *sql.DB) {
    var query string
    switch {
    case table == RES_TABLE_NAME:
        query = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
            id INTEGER PRIMARY KEY AUTOINCREMENT, 
            resource TEXT UNIQUE
        )`, RES_TABLE_NAME)
    case table == PW_TABLE_NAME:
        query = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
             id INTEGER PRIMARY KEY AUTOINCREMENT, 
             resource_id INTEGER, 
             username TEXT, 
             password TEXT, 
             type TEXT, 
             FOREIGN KEY(resource_id) REFERENCES pw_resources(id)
        )`, PW_TABLE_NAME)
    default:
        fmt.Printf("Unknown table name in createTable: %s\n", table)
        return
    }

    _, err := db.Exec(query)
    if err != nil {
        panic(err)
    }
}


func checkTables(db_name string, db *sql.DB) {
    // Create default tables if not exists
    if !isTablePresents(RES_TABLE_NAME, db) {
        createTables(RES_TABLE_NAME, db)
        fmt.Printf("created table [%s] in <%s>, done.", RES_TABLE_NAME, db_name)
    } else {
        fmt.Printf("table [%s] already in <%s>, do nothing.", RES_TABLE_NAME, db_name)
    }

    if !isTablePresents(PW_TABLE_NAME, db) {
        createTables(PW_TABLE_NAME, db)
        fmt.Printf("created table [%s] in <%s>, done.", PW_TABLE_NAME, db_name)
    } else {
        fmt.Printf("table [%s] already in <%s>, do nothing.", PW_TABLE_NAME, db_name)
    }
}


func printRes(db *sql.DB) {
    query := fmt.Sprintf("SELECT id, resource FROM %s", RES_TABLE_NAME)
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
    if err = rows.Err(); err != nil {
        panic(err.Error())
    }
}


func addPW(record *PasswordRecord, db *sql.DB) {
    var id    int 
    var query string

    if _, err := strconv.Atoi(record.resource); err == nil {
        query = fmt.Sprintf("SELECT id FROM %s WHERE id = ?", RES_TABLE_NAME)
    } else {
        query = fmt.Sprintf("SELECT id FROM %s WHERE resource = ?", RES_TABLE_NAME)
    }

    // find id for resource or create it 
    if err := db.QueryRow(query, record.resource).Scan(&id); err != nil {
        if err == sql.ErrNoRows {
            fmt.Println("no resource found, creating it..")
            id, err = insertRes(record.resource, db)
            if err != nil {
                panic(err)
            }
        } else {
            panic(err)
        }
    }
    insertPW(id, record, db)
}


func insertRes(resource string, db *sql.DB) (id int, err error) {
    query := fmt.Sprintf("INSERT INTO %s(resource) VALUES(?) RETURNING id", RES_TABLE_NAME)
    err = db.QueryRow(query, resource).Scan(&id)
    return id, err
}


func insertPW(id int, record *PasswordRecord, db *sql.DB) {
    query := fmt.Sprintf("INSERT INTO %s(resource_id, username, password, type) VALUES($1, $2, $3, $4)", PW_TABLE_NAME)
    _, err := db.Exec(query, id, record.user, record.passwd, record.app)
    if err != nil {
        panic(err.Error())
    }   
    return 
}

func showPW(db *sql.DB, arg string) {
    query := fmt.Sprintf(`SELECT R.id, P.id, R.resource, P.username, P.password, P.type 
        FROM %s R 
          LEFT JOIN %s P ON R.id=P.resource_id `, 
        RES_TABLE_NAME, PW_TABLE_NAME)

    if len(arg) != 0 {
        if _, err := strconv.Atoi(arg); err == nil {
            query += " WHERE R.id = ?"
        } else {
            query += " WHERE R.resource = ?"
        }
    }
    query += " ORDER BY R.resource"

    rows, err := db.Query(query, arg)
    if err != nil {
        panic(err)
    }
    defer rows.Close()

    type P struct {
        rid, pid         sql.NullInt64 
        res, us, pw, app sql.NullString
        pwdec            string
    }
    var Pmlens = [7]int{0, 0, 0, 0, 0, 0, 0}

    var Ps []P
    
    for rows.Next() {
        var p P
        if err := rows.Scan(&p.rid, &p.pid, &p.res, &p.us, &p.pw, &p.app); err != nil {
            panic(err)
        }
        if p.pid.Int64 == 0 {
            fmt.Printf("%d %s:\tNO CREDENTIALS\n", 
                        p.rid.Int64, p.res.String)
            continue
        }
        p.pwdec = decrypt(p.pw.String, PW_KEY)

        if Pmlens[0] < getIntStrLen(p.rid.Int64) {
            Pmlens[0] = getIntStrLen(p.rid.Int64)
        }
        if Pmlens[1] < getIntStrLen(p.pid.Int64) {
            Pmlens[1] = getIntStrLen(p.pid.Int64)
        }
        if Pmlens[2] < len(p.res.String) {
            Pmlens[2] = len(p.res.String)
        }
        if Pmlens[3] < len(p.us.String) {
            Pmlens[3] = len(p.us.String)
        }
        if Pmlens[4] < len(p.pw.String) {
            Pmlens[4] = len(p.pw.String)
        }
        if Pmlens[5] < len(p.app.String) {
            Pmlens[5] = len(p.app.String)
        }
        if Pmlens[6] < len(p.pwdec) {
            Pmlens[6] = len(p.pwdec)
        }

        Ps = append(Ps, p)
    }
    if err = rows.Err(); err != nil {
        panic(err.Error())
    }
    for _, p := range Ps {
        fmt.Printf("%*d %*s:\t%*s\t%*s\t%*s\t(%*d)\n", 
                    Pmlens[0], p.rid.Int64,  Pmlens[2], p.res.String, 
                    Pmlens[3], p.us.String,  Pmlens[6], p.pwdec, 
                    Pmlens[5], p.app.String, Pmlens[1], p.pid.Int64)
    }
    return
}

func getIntStrLen(i int64) (int){
    if i < 10 {
        return 1
    } else if i < 100 {
        return 2
    } else if i < 1000 {
        return 3
    } else if i < 10000 {
        return 4
    } else if i < 100000 {
        return 5
    } else if i < 1000000 {
        return 6
    } else if i < 10000000 {
        return 7
    }
    return 8
}

func delResource(id string, db *sql.DB) {
    queries := []string{"DELETE FROM pw_resources WHERE id = ?",
                        "DELETE FROM pw_passwords WHERE resource_id = ?"}
    for _, q := range queries {
        _, err := db.Exec(q, id) 
        if err != nil {
            panic(err)
        }
    }
    return
}

func delPassword(id string, db *sql.DB) {
    query := "DELETE FROM pw_passwords WHERE id = ?"
    _, err := db.Exec(query, id) 
    if err != nil {
        panic(err)
    }
    return
}
