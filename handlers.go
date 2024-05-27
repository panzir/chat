package main

import (
    "net/http"
    "html/template"
    "strconv"
    "fmt"
    "sync"
    //"time"
    "encoding/json"
    
    "github.com/gorilla/websocket"
    "github.com/gorilla/mux"
)

func (app *Application) SignUp(w http.ResponseWriter, r *http.Request) {
    user, err1 := r.Cookie("user")
    Username := ""
    if err1 == nil && len(user.Value) > 0 {
        Username = user.Value
    }
    
    RoomNum := -1
    room, err2 := r.Cookie("room")
    if err2 == nil {
        i, err := strconv.Atoi(room.Value)
        if err == nil {
            RoomNum = i
        }
    }
        
    //пользователь зарегистрирован в комнате -> в комнату
    if Username != "" && RoomNum >= 0 {
        http.Redirect(w, r, "/" + strconv.Itoa(RoomNum), http.StatusSeeOther)
        return
    }
    http.SetCookie(w,&http.Cookie{Name: "room", MaxAge: -1})
    
    files := []string{
        "./html/signup.page.html",
    }
    
    ts, err := template.ParseFiles(files...)
	if err != nil {
		w.WriteHeader(500)
        fmt.Fprintf(w,"Internal server error") 
		return
	}

    err = ts.Execute(w, Username)
	if err != nil {
		w.WriteHeader(500)
        fmt.Fprintf(w,"Internal server error") 
	}
}

func (app *Application) SignIn(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")
    room := r.FormValue("room")
    
    if user == "" {
        http.SetCookie(w,&http.Cookie{Name: "user", MaxAge: -1})
        http.SetCookie(w,&http.Cookie{Name: "room", MaxAge: -1})
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    } else {
        http.SetCookie(w,&http.Cookie{Name: "user", Value: user})
    }
    
    i, err := strconv.Atoi(room)
    if err == nil && i >= 0 {
        http.SetCookie(w,&http.Cookie{Name: "room", Value: room})
    }
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Application) SignOut(w http.ResponseWriter, r *http.Request) {
    http.SetCookie(w,&http.Cookie{Name: "room", MaxAge: -1})
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Application) Chat(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	
    user, err1 := r.Cookie("user")
    Username := ""
    if err1 == nil && len(user.Value) > 0 {
        Username = user.Value
    }
    
    RoomNum := -1
    room, err2 := r.Cookie("room")
    if err2 == nil {
        i, err := strconv.Atoi(room.Value)
        if err == nil {
            RoomNum = i
        }
    }
    
    if Username == "" || RoomNum < 0 || RoomNum != id {
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }
    
    files := []string{
			"./html/chat.page.html",
    }
    
    ts, err := template.ParseFiles(files...)
	if err != nil {
		w.WriteHeader(500)
        fmt.Fprintf(w,"Internal server error") 
		return
	}
    
    address := "ws://" + app.Addr + "/ws?user=" + Username + "&room=" + strconv.Itoa(RoomNum)
    
    err = ts.Execute(w, address)
	if err != nil {
		w.WriteHeader(500)
        fmt.Fprintf(w,"Internal server error") 
	}
}

func (app *Application) WS (w http.ResponseWriter, r *http.Request) {
    user := r.URL.Query().Get("user")
    room, err := strconv.Atoi(r.URL.Query().Get("room")); 
    if err != nil || user == "" {
        return
    }
    
    upgrader.CheckOrigin = func(r *http.Request) bool { return true }
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    
    var wg sync.WaitGroup
    message := make(chan MessageHeader,2)
    
    //добавление пользователя в комнату (создание комнат)
    app.mutex.Lock()
    singleRoom := Room{}
    sr, exists := app.rooms[room]
    if exists == true {
        singleRoom = sr
    } else {
        singleRoom.Visitors = make(map[string]chan MessageHeader)
    }
    singleRoom.Visitors[user] = message
    app.rooms[room] = singleRoom
    app.mutex.Unlock()
    
    wg.Add(1)
    go func() {
        for {
            m, ok := <-message
            if ok != true {
                break
            }
            mes, err := app.pg.GetMessage(m.Id)
            s := fmt.Sprintf("[%s] %s: %s", mes.Created.Format("2 Jan 2006 15:04"), mes.Username, mes.Text)
            if err != nil {
                continue
            }
            conn.WriteMessage(websocket.TextMessage, []byte(s))
        }
        wg.Done()
    }()
    
    //передача последних сообщений при входе в комнату
    headers, err := app.pg.GetLastMessages(room)
    if err == nil {
        for _, h := range(headers) {
            message <- h
        } 
    }
    
    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            //пользователь ушел - убираем из комнаты и саму комнату если нужно
            app.mutex.Lock()
            if len(app.rooms[room].Visitors) == 1 {
                delete(app.rooms, room) //удаляем комнату
            } else {
                singleRoom = app.rooms[room]
                delete(singleRoom.Visitors,user) //удаляем пользователя
                app.rooms[room] = singleRoom
            }
            app.mutex.Unlock()
            close(message)
            break
        }
        header, err := app.pg.AddMessage(room,user,string(msg))
        if err != nil {
            continue
        }
        if RABBIT {
            jsn, err := json.Marshal(header)
            if err != nil {
                continue
            }
            app.RabbitPublish(jsn)
        } else {
            app.messagePool <- header
        }
    }
    
    wg.Wait()
}
