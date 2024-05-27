package main 

import (
    "log"
    "database/sql"
    "time"
    "sync"
)

type PgModel struct {
	DB *sql.DB
}

type Message struct {
    Id int
    Username string
    Room int
    Created time.Time
    Text string
}

type MessageHeader struct {
    Id int
    Room int
}

type Room struct {
    Visitors map[string]chan MessageHeader
}

type Application struct {
	errorLog *log.Logger
	infoLog  *log.Logger
	pg *PgModel
	Addr string
	rooms map[int]Room
	mutex *sync.RWMutex
	messagePool chan MessageHeader
}

