package main

import (
	"math/rand"
	"os"
	"time"
)

func readKey() byte {
	var buf [1]byte
	os.Stdin.Read(buf[:])
	return buf[0]
}

func main() {
	rand.Seed(time.Now().UnixNano())

	for {
		game := &Game{
			Player: NewPlayer(),
		}
		game.GenerateMap()
		game.Render()

		for {
			c := readKey()
			if game.GameOver {
				if c == 'r' || c == 'R' {
					break
				} else {
					os.Exit(0)
				}
			}
			if game.Battle != nil {
				if c == 'a' || c == 'A' {
					game.BattleAttack()
				} else if c == 'f' || c == 'F' {
					game.BattleFlee()
				}
			} else {
				switch c {
				case 'w', 'W':
					game.MovePlayer(0, -1)
				case 's', 'S':
					game.MovePlayer(0, 1)
				case 'a', 'A':
					game.MovePlayer(-1, 0)
				case 'd', 'D':
					game.MovePlayer(1, 0)
				}
			}
			game.Render()
		}
	}
}
