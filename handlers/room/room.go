package room

import (
	"encoding/json"
	"log"
	"sync"
)

type Message struct {
	Playerid string `json:"playerid"`
	Text     string `json:"text"`
	MsgType  string `json:"type"`
}
type Card struct {
	Name string `json:"name"`
	Team byte   `json:"team"`
}

type Hint struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

type GameState struct {
	Roomid   string         `json:"roomid"`
	Owner    Player         `json:"owner"`
	Cards    []Card         `json:"cards"`
	RedSpy   Player         `json:"redspy"`
	BlueSpy  Player         `json:"bluespy"`
	Teamred  []Player       `json:"teamred"`
	Teamblue []Player       `json:"teamblue"`
	Gamelogs []Message      `json:"gamelogs"`
	Score    map[byte]int32 `json:"score"`
	Turn     byte           `json:"turn"`
	Hint     Hint           `json:"hint"`
}

/*
	GAME STATE:
		- roomid
		- owner
		- cards
		- redspy
		- bluespy
		- teamred
		- teamblue
		- gamelogs
		- score
		- turn
*/

type Room struct {
	GameState
	sync.RWMutex

	register   chan *Player
	unregister chan *Player
	broadcast  chan *Message
	personal   chan *Message
	players    map[*Player]bool
	anscards   []Card
}

func newRoom(roomid string, owner Player) *Room {
	log.Printf("Creating Room")
	anscard, cards := CardCategory()
	return &Room{
		players:    make(map[*Player]bool),
		register:   make(chan *Player),
		unregister: make(chan *Player),
		broadcast:  make(chan *Message),
		personal:   make(chan *Message),
		anscards:   anscard,
		GameState: GameState{
			Roomid:   roomid,
			Owner:    owner,
			Cards:    cards,
			RedSpy:   Player{},
			BlueSpy:  Player{},
			Teamred:  make([]Player, 0),
			Teamblue: make([]Player, 0),
			Gamelogs: make([]Message, 0),
			Score:    map[byte]int32{'R': 0, 'B': 0},
			Turn:     'R',
			Hint:     Hint{Word: "", Count: 0},
		},
	}
}

type Lobby struct {
	Rooms map[string]*Room
}

func NewLobby() *Lobby {
	return &Lobby{
		Rooms: make(map[string]*Room),
	}
}

func (r *Room) run() {
	for {
		select {
		case player := <-r.register:
			{
				// listening the player registering to room.
				r.Lock()
				log.Printf("Registering the player : %v", player.Nickname)
				r.players[player] = true
				r.Unlock()

				// MSG : send the game state
				// TODO:
				if r.RedSpy.Id == player.Id || r.BlueSpy.Id == player.Id {
					gs := r.GameState
					gs.Cards = r.anscards
					spyData, _ := json.Marshal(gs)
					if spyData != nil {
						player.send <- spyData
					}
				} else {
					gameState, _ := json.Marshal(r.GameState)
					player.send <- gameState
				}
			}
		case player := <-r.unregister:
			{
				// listening the player unregistering from room
				// check if client exists and remove the client entry
				r.Lock()
				if _, playerExist := r.players[player]; playerExist {
					r.players[player] = false
					close(player.send)
					log.Printf("Unregistering the player : %v", player.Nickname)
					delete(r.players, player)
				}
				r.Unlock()
			}
		case m := <-r.broadcast:
			{
				r.RLock()
				var msgData []byte
				var spyData []byte
				if m.MsgType == "INFO" {
					r.Gamelogs = append(r.Gamelogs, *m)
					msgData, _ = json.Marshal(r.GameState)
				} else if m.MsgType == "SPY" {
					r.Gamelogs = append(r.Gamelogs, *m)
					// generate new game state with ans cards
					// gs := r.GameState
					// gs.Cards = r.anscards
					// spyData, _ = json.Marshal(gs)
					msgData, _ = json.Marshal(r.GameState)
				} else {
					msgData, _ = json.Marshal(m)
				}

				for p := range r.players {
					if p.Id == r.BlueSpy.Id || p.Id == r.RedSpy.Id {
						gs := r.GameState
						gs.Cards = r.anscards
						spyData, _ = json.Marshal(gs)
						p.send <- spyData

						continue
					}
					select {
					case p.send <- msgData:
					default:
						close(p.send)
						delete(r.players, p)
					}
				}

				r.RUnlock()

			}
		}
	}
}
