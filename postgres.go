package main

import (
    //"fmt"
)

func (pg *PgModel) AddMessage(room int, username, message string) (MessageHeader, error) {   
    var header MessageHeader
    header.Room = room
    
    stmt, err := pg.DB.Prepare(`INSERT INTO messages(room,username,message) VALUES ($1,$2,$3) RETURNING id;`)
    if err != nil {
        return header, err
    }

    err = stmt.QueryRow(room,username,message).Scan(&header.Id)
    if err != nil {
        return header, err
    }

	return header, nil
} 

func (pg *PgModel) GetMessage(id int) (Message, error) {   
	var m Message
	
	stmt := `SELECT id, username, room, created_at, message FROM messages WHERE id = $1;`
	err := pg.DB.QueryRow(stmt, id).Scan(&m.Id, &m.Username, &m.Room, &m.Created, &m.Text)
    if err != nil {
        return m, err
    }
    
	return m, nil
} 

func (pg *PgModel) GetLastMessages(room int) ([]MessageHeader, error) {   
	var headers []MessageHeader
	stmt := `SELECT * FROM (SELECT id FROM messages WHERE room = $1 ORDER BY id DESC LIMIT 50) ORDER BY id ASC`
	
	rows, err := pg.DB.Query(stmt, room)
	if err != nil {
		return headers, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var h MessageHeader
		h.Room = room
		if err = rows.Scan(&h.Id); err != nil {
			return headers, err
		}
		headers = append(headers, h)
	}
	if err = rows.Err(); err != nil {
		return headers, err
	}
	return headers, nil
} 
