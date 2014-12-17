package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Match struct {
	Score   Score    `json:"score"`
	Players []Player `json:"players"`
}

type Score struct {
	Overall     BothHalves `json:"overall"`
	FirstHalf   Half       `json:"firsthalf"`
	SecondHalf  Half       `json:"secondhalf"`
	WinSequence []string   `json:"winsequence"`
}

type BothHalves struct {
	Terrorists        uint `json:"T"`
	CounterTerrorists uint `json:"CT"`
}

type Half struct {
	Terrorists        uint `json:"terrorists"`
	CounterTerrorists uint `json:"counterterrorists"`
}

type Player struct {
	Played      bool   `json:"-"`
	MVPs        uint   `json:"mvps"`
	Nickname    string `json:"nickname"`
	SteamID     string `json:"steamid"`
	UserID      string `json:"-"`
	InitialSide string `json:"team"`
}

// WriteFile writes given buffer to given filename. It creates the file
// it is not yet created. The created file uses 0600 permissions by default.
func writeFile(filename string, buffer bytes.Buffer) error {

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer f.Close()
	if _, err = f.WriteString(buffer.String()); err != nil {
		return err
	}

	return nil
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func format(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "userID:")
	line = strings.TrimPrefix(line, "name:")
	line = strings.TrimPrefix(line, "guid:")
	line = strings.TrimPrefix(line, "team:")
	line = strings.TrimSpace(line)
	return line
}

func formatMVP(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "userid:")
	spaces := strings.Split(line, " ")
	line = strings.Join(spaces[:len(spaces)-1], " ")
	line = strings.TrimSpace(line)
	return line
}

func formatRounds(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "winner:")
	line = strings.TrimSpace(line)
	switch line {
	case "2":
		return "T"
	case "3":
		return "CT"
	}
	return "?"
}

func (match Match) countRounds() Match {
	for index, winner := range match.Score.WinSequence {
		if index < 15 {
			if winner == "T" {
				match.Score.FirstHalf.Terrorists += 1
			}
			if winner == "CT" {
				match.Score.FirstHalf.CounterTerrorists += 1
			}
		} else {
			if winner == "T" {
				match.Score.SecondHalf.Terrorists += 1
			}
			if winner == "CT" {
				match.Score.SecondHalf.CounterTerrorists += 1
			}
		}
	}
	match.Score.Overall.Terrorists = match.Score.FirstHalf.Terrorists + match.Score.SecondHalf.CounterTerrorists
	match.Score.Overall.CounterTerrorists = match.Score.FirstHalf.CounterTerrorists + match.Score.SecondHalf.Terrorists
	return match
}

func main() {
	var match Match

	lines, err := readLines(os.Args[1])
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(lines); i++ {
		if lines[i] == "adding:player info:" && lines[i+4] != " guid:BOT" {
			var player Player
			player.Nickname = format(lines[i+2])
			player.SteamID = format(lines[i+4])
			player.UserID = format(lines[i+3])
			player.UserID = fmt.Sprintf(` userid: %s (id:%s)`, player.Nickname, player.UserID)
			match.Players = append(match.Players, player)
		}
		if strings.Contains(lines[i], " userid: ") {
			for index, _ := range match.Players {
				if match.Players[index].UserID == lines[i] && !match.Players[index].Played && lines[i-2] == "weapon_fire" {
					match.Players[index].Played = true
					match.Players[index].InitialSide = format(lines[i+3])
				}
			}
		}
		if lines[i] == "round_mvp" {
			nickname := formatMVP(lines[i+2])
			for i, player := range match.Players {
				if player.Nickname == nickname {
					match.Players[i].MVPs += 1
				}
			}
		}
		if lines[i] == "round_end" {
			match.Score.WinSequence = append(match.Score.WinSequence, formatRounds(lines[i+2]))
		}
	}

	for i := 0; i < len(match.Players); i++ {
		if !match.Players[i].Played {
			match.Players[i] = match.Players[len(match.Players)-1]
			match.Players = match.Players[0 : len(match.Players)-1]
		}
	}

	match = match.countRounds()

	matchJSON, err := json.Marshal(match)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fmt.Sprintf("ids-%s", os.Args[1]), matchJSON, 0600)
	if err != nil {
		panic(err)
	}
}
