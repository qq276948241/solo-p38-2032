package main

import (
	"fmt"
	"math/rand"
)

func (g *Game) GenerateMap() {
	g.Map = make([][]byte, MapHeight)
	g.Visited = make([][]bool, MapHeight)
	for y := range g.Map {
		g.Map[y] = make([]byte, MapWidth)
		g.Visited[y] = make([]bool, MapWidth)
		for x := range g.Map[y] {
			g.Map[y][x] = TileWall
		}
	}

	rooms := g.generateRooms()
	g.connectRooms(rooms)

	if len(rooms) > 0 {
		g.Player.X, g.Player.Y = rooms[0].Center()
		g.Map[g.Player.Y][g.Player.X] = TileFloor
	}

	if len(rooms) > 1 {
		last := rooms[len(rooms)-1]
		g.Exit = [2]int{last.CenterX(), last.CenterY()}
		g.Map[g.Exit[1]][g.Exit[0]] = TileExit
	}

	g.PlaceEntities(rooms)
	g.UpdateVisibility()
}

func (g *Game) UpdateVisibility() {
	px, py := g.Player.X, g.Player.Y
	for dy := -VisionRadius; dy <= VisionRadius; dy++ {
		for dx := -VisionRadius; dx <= VisionRadius; dx++ {
			if dx*dx+dy*dy <= VisionRadius*VisionRadius {
				nx, ny := px+dx, py+dy
				if nx >= 0 && nx < MapWidth && ny >= 0 && ny < MapHeight {
					g.Visited[ny][nx] = true
				}
			}
		}
	}
}

func (g *Game) InVision(x, y int) bool {
	dx := x - g.Player.X
	dy := y - g.Player.Y
	return dx*dx+dy*dy <= VisionRadius*VisionRadius
}

func (g *Game) generateRooms() []Room {
	var rooms []Room
	maxAttempts := 100
	targetRooms := 6 + rand.Intn(4)

	for i := 0; i < maxAttempts && len(rooms) < targetRooms; i++ {
		w := 4 + rand.Intn(7)
		h := 3 + rand.Intn(5)
		x := 1 + rand.Intn(MapWidth-w-2)
		y := 1 + rand.Intn(MapHeight-h-2)
		newRoom := Room{x, y, w, h}

		ok := true
		for _, r := range rooms {
			if newRoom.Intersects(r) {
				ok = false
				break
			}
		}
		if ok {
			rooms = append(rooms, newRoom)
			for yy := y; yy < y+h; yy++ {
				for xx := x; xx < x+w; xx++ {
					g.Map[yy][xx] = TileFloor
				}
			}
		}
	}
	return rooms
}

func (g *Game) connectRooms(rooms []Room) {
	for i := 1; i < len(rooms); i++ {
		ax, ay := rooms[i-1].Center()
		bx, by := rooms[i].Center()
		if rand.Intn(2) == 0 {
			g.carveHCorridor(ax, bx, ay)
			g.carveVCorridor(ay, by, bx)
		} else {
			g.carveVCorridor(ay, by, ax)
			g.carveHCorridor(ax, bx, by)
		}
	}
}

func (g *Game) carveHCorridor(x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		if y > 0 && y < MapHeight-1 && x > 0 && x < MapWidth-1 {
			g.Map[y][x] = TileFloor
		}
	}
}

func (g *Game) carveVCorridor(y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if y > 0 && y < MapHeight-1 && x > 0 && x < MapWidth-1 {
			g.Map[y][x] = TileFloor
		}
	}
}

func (g *Game) PlaceEntities(rooms []Room) {
	g.Monsters = nil
	g.Potions = make(map[[2]int]int)
	g.Golds = make(map[[2]int]int)

	floor := g.Player.Floor
	monsterCount := 3 + floor + rand.Intn(3)
	potionCount := 2 + rand.Intn(2)
	goldCount := 4 + rand.Intn(4)

	for i := 0; i < monsterCount; i++ {
		if m := g.spawnMonster(rooms, floor); m != nil {
			g.Monsters = append(g.Monsters, m)
		}
	}

	for i := 0; i < potionCount; i++ {
		g.placeRandomItem(rooms, func(x, y int) {
			heal := 8 + rand.Intn(8) + floor*2
			g.Potions[[2]int{x, y}] = heal
		})
	}

	for i := 0; i < goldCount; i++ {
		g.placeRandomItem(rooms, func(x, y int) {
			amount := 5 + rand.Intn(10) + floor*3
			g.Golds[[2]int{x, y}] = amount
		})
	}

	if floor == MaxFloor {
		g.placeBoss(rooms)
	}
}

func (g *Game) spawnMonster(rooms []Room, floor int) *Monster {
	for attempts := 0; attempts < 50; attempts++ {
		if len(rooms) < 2 {
			return nil
		}
		r := rooms[1+rand.Intn(len(rooms)-1)]
		x := r.X + rand.Intn(r.W)
		y := r.Y + rand.Intn(r.H)
		if g.isSpawnable(x, y) {
			hp := 8 + floor*3 + rand.Intn(5)
			return &Monster{
				X:     x,
				Y:     y,
				HP:    hp,
				MaxHP: hp,
				Atk:   2 + floor + rand.Intn(3),
				Def:   1 + rand.Intn(floor),
				Gold:  5 + rand.Intn(8) + floor*2,
				Name:  randomMonsterName(),
			}
		}
	}
	return nil
}

func (g *Game) placeBoss(rooms []Room) {
	for attempts := 0; attempts < 50; attempts++ {
		if len(rooms) < 1 {
			return
		}
		r := rooms[len(rooms)-1]
		x := r.CenterX()
		y := r.CenterY()
		if (g.Map[y][x] == TileFloor || g.Map[y][x] == TileExit) && !g.hasEntity(x, y) {
			hp := 60
			boss := &Monster{
				X:     x,
				Y:     y,
				HP:    hp,
				MaxHP: hp,
				Atk:   12,
				Def:   5,
				Gold:  100,
				Name:  "地牢领主",
				Boss:  true,
			}
			g.Monsters = append(g.Monsters, boss)
			g.Exit = [2]int{-1, -1}
			return
		}
	}
}

func (g *Game) placeRandomItem(rooms []Room, setter func(x, y int)) {
	for attempts := 0; attempts < 50; attempts++ {
		if len(rooms) < 2 {
			return
		}
		r := rooms[1+rand.Intn(len(rooms)-1)]
		x := r.X + rand.Intn(r.W)
		y := r.Y + rand.Intn(r.H)
		if g.isSpawnable(x, y) {
			setter(x, y)
			return
		}
	}
}

func (g *Game) isSpawnable(x, y int) bool {
	if g.Map[y][x] != TileFloor {
		return false
	}
	return !g.hasEntity(x, y)
}

func (g *Game) hasEntity(x, y int) bool {
	if x == g.Player.X && y == g.Player.Y {
		return true
	}
	for _, m := range g.Monsters {
		if m.X == x && m.Y == y {
			return true
		}
	}
	if _, ok := g.Potions[[2]int{x, y}]; ok {
		return true
	}
	if _, ok := g.Golds[[2]int{x, y}]; ok {
		return true
	}
	if g.Exit[0] == x && g.Exit[1] == y {
		return true
	}
	return false
}

func randomMonsterName() string {
	names := []string{"史莱姆", "哥布林", "骷髅兵", "蝙蝠", "野狼", "食人妖", "暗影", "僵尸"}
	return names[rand.Intn(len(names))]
}

func (g *Game) NextFloor() {
	g.Player.Floor++
	if g.Player.Floor > g.Player.MaxFloor {
		g.Player.MaxFloor = g.Player.Floor
	}
	g.Msg = fmt.Sprintf("进入了第 %d 层！", g.Player.Floor)
	g.GenerateMap()
}
