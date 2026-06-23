package main

const (
	MapWidth     = 60
	MapHeight    = 20
	MaxFloor     = 5
	VisionRadius = 5
	TileWall     = '#'
	TileFloor    = '.'
	TilePlayer   = '@'
	TileMonster  = 'M'
	TilePotion   = '+'
	TileGold     = '$'
	TileExit     = '>'
	TileBoss     = 'B'
)

const (
	csiReset  = "\x1b[0m"
	csiFgGray = "\x1b[90m"
)

type Player struct {
	X, Y     int
	HP       int
	MaxHP    int
	Atk      int
	Def      int
	Gold     int
	Floor    int
	MaxFloor int
}

type Monster struct {
	X, Y  int
	HP    int
	MaxHP int
	Atk   int
	Def   int
	Gold  int
	Name  string
	Boss  bool
}

type Game struct {
	Map      [][]byte
	Visited  [][]bool
	Player   Player
	Monsters []*Monster
	Potions  map[[2]int]int
	Golds    map[[2]int]int
	Exit     [2]int
	Msg      string
	Battle   *Battle
	GameOver bool
	Win      bool
}

type Battle struct {
	Monster *Monster
	Log     []string
}

type Room struct {
	X, Y, W, H int
}

func (r Room) CenterX() int               { return r.X + r.W/2 }
func (r Room) CenterY() int               { return r.Y + r.H/2 }
func (r Room) Center() (int, int)         { return r.CenterX(), r.CenterY() }
func (r Room) Intersects(o Room) bool {
	return r.X <= o.X+o.W+1 && r.X+r.W+1 >= o.X &&
		r.Y <= o.Y+o.H+1 && r.Y+r.H+1 >= o.Y
}

func NewPlayer() Player {
	return Player{
		HP:       30,
		MaxHP:    30,
		Atk:      5,
		Def:      2,
		Gold:     0,
		Floor:    1,
		MaxFloor: 1,
	}
}
