package room

import (
	"codenames-server/utils"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var connUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
		// origin := r.Header.Get("Origin")
		// log.Print(origin)
		// TODO: check for origins
	},
}

var LobbyList = NewLobby()

// handle creation of room-id and initializing it and starting the room go-routine
func CreateRoom(c echo.Context) error {

	roomid := c.Request().URL.Query().Get("roomid")
	nickname := c.Request().URL.Query().Get("nickname")
	playerid := c.Request().URL.Query().Get("playerid")

	if roomid == "" || nickname == "" || playerid == "" {
		log.Printf("Roomid: %v\nnickname: %v", roomid, nickname)
		log.Printf("Room or User ids are invalid")
		return c.JSON(http.StatusBadRequest, utils.Res{Message: "Room or User is not mentioned", Ok: false})
	}
	nr := newRoom(roomid, Player{Id: playerid, Nickname: nickname})
	LobbyList.Rooms[roomid] = nr
	go nr.run()
	return c.JSON(http.StatusOK, utils.Res{Message: "Room created", Ok: true})
}

func JoinRoom(c echo.Context) error {
	roomid := c.Request().URL.Query().Get("roomid")
	nickname := c.Request().URL.Query().Get("nickname")
	playerid := c.Request().URL.Query().Get("playerid")

	if roomid == "" || nickname == "" || playerid == "" {
		log.Printf("Roomid: %v\nnickname: %v", roomid, nickname)
		log.Printf("Room or User ids are invalid")
		return c.JSON(http.StatusBadRequest, utils.Res{Message: "Room or User is not mentioned", Ok: false})
	}

	conn, wsErr := connUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if wsErr != nil {
		log.Printf("Error while upgrading http connection to websocket")
		log.Print(wsErr)
	}

	/*
		1. if roomid exists or not
		2. if not throw the error
		3. if yes -> register the client in that room
		4. and send the data to user. (only currated data not all)s
	*/

	room, ok := LobbyList.Rooms[roomid]
	if !ok {
		log.Printf("Room does not exists with id: " + roomid)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room not found"))
		return conn.Close()
		// conn.Close()
		// return c.JSON(http.StatusNotFound, utils.Res{Message: "Room not found", Ok: false})
	}
	log.Printf("%+v", room)
	if room == nil {
		log.Printf("Room does not exists with id: " + roomid)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Room not found"))
		return conn.Close()
		// return c.JSON(http.StatusNotFound, utils.Res{Message: "Room not found", Ok: false})
	}

	// room present

	newPlayer := &Player{Id: playerid, Nickname: nickname, room: room, conn: conn, send: make(chan []byte, 512)}
	newPlayer.room.register <- newPlayer

	log.Printf("Player joined the room")
	joinMsg := &Message{Playerid: playerid, MsgType: "MSG", Text: fmt.Sprintf("%v Joined the room", nickname)}
	newPlayer.room.broadcast <- joinMsg

	// now start the goroutines for read and write the data
	go newPlayer.writePump()
	go newPlayer.readPump()
	return nil
}
