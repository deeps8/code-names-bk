package room

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func PlayerExistInTeam(team []Player, player Player) bool {
	for _, p := range team {
		if p.Id == player.Id {
			return true
		}
	}
	return false
}

func RemovePlayerFromTeam(team []Player, player Player) []Player {
	for i, p := range team {
		if p.Id == player.Id {
			team = append(team[:i], team[i+1:]...)
			break
		}
	}
	return team
}

func CardCategory() ([]Card, []Card) {

	res, err := http.Get("https://random-word-form.herokuapp.com/random/noun?count=25")
	if err != nil {
		log.Fatal(err)
		return nil, nil
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Error reading response body:", err)
		return nil, nil
	}

	// Parse the JSON response into a slice of strings
	var nouns []string
	err = json.Unmarshal(body, &nouns)
	if err != nil {
		log.Fatal("Error parsing JSON:", err)
		return nil, nil
	}

	// Total number of cards
	totalCards := 25

	// Card distribution
	numRed := 9   // Starting team
	numBlue := 8  // Other team
	numBlack := 1 // Assassin
	// numGrey := 7  // Bystanders

	// Create a slice of indices from 0 to totalCards-1
	cards := make([]Card, totalCards)
	nums := make([]int, totalCards)
	for i := 0; i < totalCards; i++ {
		cards[i] = Card{Name: nouns[i], Team: 'O'}
		nums[i] = i
	}

	// Shuffle the indices randomly
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(nums), func(i, j int) {
		nums[i], nums[j] = nums[j], nums[i]
	})

	// Distribute cards
	anscards := append([]Card{}, cards...)
	for i := 0; i < totalCards; i++ {
		if i < numRed {
			anscards[nums[i]].Team = 'R'
		} else if i < numRed+numBlue {
			anscards[nums[i]].Team = 'B'
		} else if i < numRed+numBlue+numBlack {
			anscards[nums[i]].Team = 'A'
		} else {
			anscards[nums[i]].Team = 'G'
		}
	}

	return anscards, cards
}
