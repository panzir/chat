package main
 
import (
	"github.com/gorilla/mux"
)

func (app *Application) routes() *mux.Router {
    
    router := mux.NewRouter()
    router.StrictSlash(true)
    router.HandleFunc("/", app.SignUp).Methods("GET")
    router.HandleFunc("/", app.SignIn).Methods("POST")
    router.HandleFunc("/signout", app.SignOut).Methods("GET")
    router.HandleFunc("/{id:[0-9]+}/", app.Chat).Methods("GET")
    
    router.HandleFunc("/ws", app.WS)
    
    return router
}
