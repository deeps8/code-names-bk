package room

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
	room     *Room
	conn     *websocket.Conn
	send     chan []byte
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 1024 * 1024
)

/*
writePump pumps the messages from hub to websocket connection
goroutine is started for each connection
*/
func (c *Player) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			{
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					c.conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}

				w, err := c.conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				log.Printf("\n%+v \n", string(msg))
				w.Write(msg)

				if err := w.Close(); err != nil {
					return
				}

			outerLoop:
				for {
					select {
					case msg, ok := <-c.send:
						if !ok {
							return
						}
						c.conn.SetWriteDeadline(time.Now().Add(writeWait))
						w, err := c.conn.NextWriter(websocket.TextMessage)
						if err != nil {
							return
						}
						w.Write(msg)
						if err := w.Close(); err != nil {
							return
						}
					default:
						break outerLoop
					}
				}
			}
		case <-ticker.C:
			{
				c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}

// readPump pumps data from websocket connection to hub
func (p *Player) readPump() {
	defer func() {
		logMsg := &Message{Playerid: p.Id, Text: fmt.Sprintf("User Left %s", p.Nickname), MsgType: "INFO"}
		p.room.broadcast <- logMsg

		p.room.unregister <- p
		p.conn.Close()
	}()

	p.conn.SetReadLimit(maxMessageSize)
	p.conn.SetReadDeadline(time.Now().Add(pongWait))
	p.conn.SetPongHandler(func(appData string) error {
		p.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, text, err := p.conn.ReadMessage()
		// log.Printf("(message type : %v)  value : %v", msgType, text)

		if err != nil {
			log.Printf("Readpump error : %v", err.Error())
			// if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// 	log.Panicf("Connection closed with err : %v", err.Error())
			// }
			break
		}

		if string(text) == "con-closed" {
			p.room.unregister <- p
			p.conn.Close()
			return
		}

		type temp struct {
			Data struct {
				Idx   float64 `json:"idx"`
				Word  string  `json:"word"`
				Count int     `json:"count"`
			} `json:"data"`
			MsgType string `json:"msgType"`
		}
		msg := temp{}
		err = json.Unmarshal(text, &msg)
		if err != nil {
			log.Printf("Error while unmarshalling the message : %v", err.Error())
			continue
		}
		log.Printf("msg : %+v", msg)

		/*
			conditions to join red team:
			1. if player already present in red team array don't add it again
			2. if player is red spy then remove from red spy and add it to red team
			3. if player is blue spy then remove from blue spy and add it to red team
			4. if player is blue team then remove from blue team and add it to red team
		*/
		if msg.MsgType == "join-red" {
			msg := "joined RED team"
			if PlayerExistInTeam(p.room.Teamred, Player{Id: p.Id}) {
				continue
			}
			if p.room.RedSpy.Id == p.Id {
				p.room.RedSpy = Player{}
				msg = "joined RED team and removed from RED SPY"
			}
			if p.room.BlueSpy.Id == p.Id {
				p.room.BlueSpy = Player{}
				msg = "joined RED team and removed from BLUE SPY"
			}
			if PlayerExistInTeam(p.room.Teamblue, Player{Id: p.Id}) {
				p.room.Teamblue = RemovePlayerFromTeam(p.room.Teamblue, Player{Id: p.Id})
				msg = "joined RED team and removed from BLUE team"
			}
			p.room.Teamred = append(p.room.Teamred, Player{Id: p.Id, Nickname: p.Nickname})
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v %v", p.Nickname, msg), Playerid: p.Id, MsgType: "INFO"}
			continue
		}

		if msg.MsgType == "join-blue" {
			msg := "joined BLUE team"
			if PlayerExistInTeam(p.room.Teamblue, Player{Id: p.Id}) {
				continue
			}
			if p.room.BlueSpy.Id == p.Id {
				p.room.BlueSpy = Player{}
				msg = "joined BLUE team and removed from BLUE SPY"
			}
			if p.room.RedSpy.Id == p.Id {
				p.room.RedSpy = Player{}
				msg = "joined BLUE team and removed from RED SPY"
			}
			if PlayerExistInTeam(p.room.Teamred, Player{Id: p.Id}) {
				p.room.Teamred = RemovePlayerFromTeam(p.room.Teamred, Player{Id: p.Id})
				msg = "joined BLUE team and removed from RED team"
			}
			p.room.Teamblue = append(p.room.Teamblue, Player{Id: p.Id, Nickname: p.Nickname})
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v %v", p.Nickname, msg), Playerid: p.Id, MsgType: "INFO"}
			continue
		}

		if msg.MsgType == "join-red-spy" {
			msg := "joined RED SPY"
			if p.room.RedSpy.Id == p.Id {
				continue
			}
			if PlayerExistInTeam(p.room.Teamred, Player{Id: p.Id}) {
				p.room.Teamred = RemovePlayerFromTeam(p.room.Teamred, Player{Id: p.Id})
				msg = "joined RED SPY and removed from RED team"
			}
			if p.room.BlueSpy.Id == p.Id {
				p.room.BlueSpy = Player{}
				msg = "joined RED SPY and removed from BLUE SPY"
			}
			if PlayerExistInTeam(p.room.Teamblue, Player{Id: p.Id}) {
				p.room.Teamblue = RemovePlayerFromTeam(p.room.Teamblue, Player{Id: p.Id})
				msg = "joined RED SPY and removed from BLUE team"
			}
			p.room.RedSpy = Player{Id: p.Id, Nickname: p.Nickname}
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v %v", p.Nickname, msg), Playerid: p.Id, MsgType: "SPY"}
			continue
		}

		if msg.MsgType == "join-blue-spy" {
			msg := "joined BLUE SPY"
			if p.room.BlueSpy.Id == p.Id {
				continue
			}
			if PlayerExistInTeam(p.room.Teamblue, Player{Id: p.Id}) {
				p.room.Teamblue = RemovePlayerFromTeam(p.room.Teamblue, Player{Id: p.Id})
				msg = "joined BLUE SPY and removed from BLUE team"
			}
			if p.room.RedSpy.Id == p.Id {
				p.room.RedSpy = Player{}
				msg = "joined BLUE SPY and removed from RED SPY"
			}
			if PlayerExistInTeam(p.room.Teamred, Player{Id: p.Id}) {
				p.room.Teamred = RemovePlayerFromTeam(p.room.Teamred, Player{Id: p.Id})
				msg = "joined BLUE SPY and removed from RED team"
			}
			p.room.BlueSpy = Player{Id: p.Id, Nickname: p.Nickname}
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v %v", p.Nickname, msg), Playerid: p.Id, MsgType: "SPY"}
			continue
		}

		if msg.MsgType == "card-click" {
			idx := int8(msg.Data.Idx)
			// team := int8(msg.Data.(temp)["team"])
			p.room.Cards[idx].Team = p.room.anscards[idx].Team
			if p.room.Cards[idx].Team == 'R' && p.room.Hint.Count > 0 {
				p.room.Score['R']++
				if p.room.Turn != 'R' {
					p.room.Hint.Count = 0
				} else {
					p.room.Hint.Count--
				}
			} else if p.room.Cards[idx].Team == 'B' && p.room.Hint.Count > 0 {
				p.room.Score['B']++
				if p.room.Turn != 'B' {
					p.room.Hint.Count = 0
				} else {
					p.room.Hint.Count--
				}
			} else {
				p.room.Hint.Count = 0
				p.room.Hint.Word = ""
			}

			if p.room.Turn == 'R' && p.room.Hint.Count == 0 {
				p.room.Turn = 'B'
				p.room.Hint.Word = ""
			} else if p.room.Turn == 'B' && p.room.Hint.Count == 0 {
				p.room.Turn = 'R'
				p.room.Hint.Word = ""
			}
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v clicked on %v", p.Nickname, p.room.Cards[idx].Name), Playerid: p.Id, MsgType: "INFO"}
			continue
		}

		if msg.MsgType == "hint" {
			p.room.Hint.Word = msg.Data.Word
			p.room.Hint.Count = msg.Data.Count
			p.room.broadcast <- &Message{Text: fmt.Sprintf("%v gave hint %v %v", p.Nickname, msg.Data.Word, msg.Data.Count), Playerid: p.Id, MsgType: "INFO"}
			continue
		}

		// msg := &Message{}
		// log.Printf("%v", text)
		// reader := bytes.NewReader(text)
		// decoder := json.NewDecoder(reader)
		// dErr := decoder.Decode(msg)
		// if dErr != nil {
		// 	log.Panicf("error while decoding msg : %v", dErr.Error())
		// }

		p.room.broadcast <- &Message{Text: string(text), Playerid: p.Id, MsgType: "MSG"}
	}
}
