package main

import (
    "log"
    "net/http"
    "flag"
    "os"
    "fmt"
    "sync"
    
    "database/sql"
	_ "github.com/lib/pq"
    
    "github.com/gorilla/websocket"
    //"github.com/joho/godotenv"
)

const (
    HOST_PG     = "172.17.0.1"
    PORT_PG     = "5454"
    USER_PG     = "pgmaster"
    PASSWORD_PG = "qwerty"
    DBNAME_PG   = "pg"
    
    RABBIT = true
    HOST_RABBIT = "172.17.0.1:5672"
)

var upgrader = websocket.Upgrader{
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}

func main() {
    addr := flag.String("addr", "127.0.0.1:8000", "Сетевой адрес веб-сервера")
	flag.Parse()
    
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
    
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", 
        HOST_PG, PORT_PG, USER_PG, PASSWORD_PG, DBNAME_PG) 
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		errorLog.Fatal(err)
	}	
	infoLog.Printf("Подключились к БД %s", DBNAME_PG)
    
    var mutex = &sync.RWMutex{}
    
    app := &Application{
		errorLog: errorLog,
		infoLog:  infoLog,
		pg: &PgModel{DB: db},
		Addr: *addr,
        mutex: mutex,
	}
	app.rooms = make(map[int]Room)
    app.messagePool = make(chan MessageHeader, 100)
    
    //обработка сообщений
    for w := 1; w <= 3; w++ {
        go func() {
            for {           
                header, ok := <-app.messagePool
                if ok != true {
                    break
                }
                app.mutex.RLock()
                //если комнаты нет
                _, exists := app.rooms[header.Room]
                if exists == true {
                    for _, ch := range app.rooms[header.Room].Visitors {
                        ch <- header
                    }
                }
                app.mutex.RUnlock()
            }
        }()
    }
    
    if RABBIT {
        go app.RabbitReceiver()
    }
    
    srv := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
	}
    
    infoLog.Printf("Запуск сервера на %s", *addr)
    errorLog.Fatal(srv.ListenAndServe())
    close(app.messagePool)
}

