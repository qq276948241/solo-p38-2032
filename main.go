package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

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

var clear map[string]func()

func init() {
	clear = make(map[string]func())
	clear["linux"] = func() {
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	enableWindowsANSI()
}

const (
	stdOutputHandle  = ^uintptr(10) + 1 // (DWORD)-11
	enableVirtualTerminalProcessing = 0x0004
)

func enableWindowsANSI() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getStdHandle := kernel32.NewProc("GetStdHandle")

	h, _, _ := getStdHandle.Call(uintptr(stdOutputHandle))
	var mode uint32
	getConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	mode |= enableVirtualTerminalProcessing
	setConsoleMode.Call(h, uintptr(mode))
}

const (
	csiReset  = "\x1b[0m"
	csiFgGray = "\x1b[90m"
)

func ClearScreen() {
	if f, ok := clear["windows"]; ok {
		f()
	} else if f, ok := clear["linux"]; ok {
		f()
	}
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

type Room struct {
	X, Y, W, H int
}

func (r Room) CenterX() int { return r.X + r.W/2 }
func (r Room) CenterY() int { return r.Y + r.H/2 }
func (r Room) Center() (int, int) { return r.CenterX(), r.CenterY() }
func (r Room) Intersects(o Room) bool {
	return r.X <= o.X+o.W+1 && r.X+r.W+1 >= o.X &&
		r.Y <= o.Y+o.H+1 && r.Y+r.H+1 >= o.Y
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
		if m.X == x && m.Y == y && m.HP > 0 {
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

func (g *Game) MovePlayer(dx, dy int) {
	if g.Battle != nil || g.GameOver {
		return
	}
	nx := g.Player.X + dx
	ny := g.Player.Y + dy
	if nx < 0 || nx >= MapWidth || ny < 0 || ny >= MapHeight {
		return
	}
	tile := g.Map[ny][nx]
	if tile == TileWall {
		return
	}

	for _, m := range g.Monsters {
		if m.X == nx && m.Y == ny && m.HP > 0 {
			g.startBattle(m)
			return
		}
	}

	g.Player.X = nx
	g.Player.Y = ny
	g.Msg = ""
	g.UpdateVisibility()

	if heal, ok := g.Potions[[2]int{nx, ny}]; ok {
		g.Player.HP = min(g.Player.MaxHP, g.Player.HP+heal)
		g.Msg = fmt.Sprintf("喝下药水，恢复了 %d 点 HP！", heal)
		delete(g.Potions, [2]int{nx, ny})
	}
	if amount, ok := g.Golds[[2]int{nx, ny}]; ok {
		g.Player.Gold += amount
		g.Msg = fmt.Sprintf("捡到了 %d 金币！", amount)
		delete(g.Golds, [2]int{nx, ny})
	}

	if g.Exit[0] == nx && g.Exit[1] == ny {
		g.NextFloor()
	}
}

func (g *Game) NextFloor() {
	g.Player.Floor++
	if g.Player.Floor > g.Player.MaxFloor {
		g.Player.MaxFloor = g.Player.Floor
	}
	g.Msg = fmt.Sprintf("进入了第 %d 层！", g.Player.Floor)
	g.GenerateMap()
}

func (g *Game) startBattle(m *Monster) {
	g.Battle = &Battle{
		Monster: m,
		Log:     []string{fmt.Sprintf("遭遇了 %s！HP:%d ATK:%d DEF:%d", m.Name, m.HP, m.Atk, m.Def)},
	}
}

func (g *Game) BattleAttack() {
	if g.Battle == nil {
		return
	}
	m := g.Battle.Monster
	pDmg := max(1, g.Player.Atk-m.Def)
	m.HP -= pDmg
	g.Battle.Log = append(g.Battle.Log, fmt.Sprintf("你造成了 %d 点伤害！", pDmg))

	if m.HP <= 0 {
		g.Battle.Log = append(g.Battle.Log, fmt.Sprintf("%s 被击败了！获得 %d 金币。", m.Name, m.Gold))
		g.Player.Gold += m.Gold
		if m.Boss {
			g.Win = true
			g.GameOver = true
		}
		g.Battle = nil
		g.Msg = fmt.Sprintf("击败了 %s！", m.Name)
		return
	}

	mDmg := max(1, m.Atk-g.Player.Def)
	g.Player.HP -= mDmg
	g.Battle.Log = append(g.Battle.Log, fmt.Sprintf("%s 造成了 %d 点伤害！", m.Name, mDmg))

	if g.Player.HP <= 0 {
		g.Player.HP = 0
		g.GameOver = true
		g.Battle.Log = append(g.Battle.Log, "你倒下了...")
	}
}

func (g *Game) BattleFlee() {
	if g.Battle == nil {
		return
	}
	m := g.Battle.Monster
	if m.Boss {
		g.Battle.Log = append(g.Battle.Log, "无法从 Boss 战中逃跑！")
		mDmg := max(1, m.Atk-g.Player.Def)
		g.Player.HP -= mDmg
		g.Battle.Log = append(g.Battle.Log, fmt.Sprintf("%s 攻击了逃跑中的你，造成 %d 点伤害！", m.Name, mDmg))
		if g.Player.HP <= 0 {
			g.Player.HP = 0
			g.GameOver = true
		}
		return
	}
	if rand.Intn(2) == 0 {
		g.Battle.Log = append(g.Battle.Log, "逃跑成功！")
		g.Battle = nil
		g.Msg = "成功逃跑了"
	} else {
		g.Battle.Log = append(g.Battle.Log, "逃跑失败！")
		mDmg := max(1, m.Atk-g.Player.Def)
		g.Player.HP -= mDmg
		g.Battle.Log = append(g.Battle.Log, fmt.Sprintf("%s 追击造成 %d 点伤害！", m.Name, mDmg))
		if g.Player.HP <= 0 {
			g.Player.HP = 0
			g.GameOver = true
		}
	}
}

func (g *Game) Render() {
	ClearScreen()

	if g.Battle != nil {
		g.RenderBattle()
		return
	}
	if g.GameOver {
		g.RenderGameOver()
		return
	}

	screen := make([][]byte, MapHeight)
	for y := range g.Map {
		screen[y] = make([]byte, MapWidth)
		copy(screen[y], g.Map[y])
	}

	for pos := range g.Potions {
		screen[pos[1]][pos[0]] = TilePotion
	}
	for pos := range g.Golds {
		screen[pos[1]][pos[0]] = TileGold
	}
	for _, m := range g.Monsters {
		if m.HP > 0 {
			if m.Boss {
				screen[m.Y][m.X] = TileBoss
			} else {
				screen[m.Y][m.X] = TileMonster
			}
		}
	}
	screen[g.Player.Y][g.Player.X] = TilePlayer

	for y := 0; y < MapHeight; y++ {
		var lineBuf []byte
		prevGray := false
		for x := 0; x < MapWidth; x++ {
			inVis := g.InVision(x, y)
			visited := g.Visited[y][x]

			var ch byte
			gray := false

			switch {
			case inVis:
				ch = screen[y][x]
			case visited:
				ch = screen[y][x]
				if ch == TileMonster || ch == TileBoss || ch == TilePotion || ch == TileGold {
					ch = g.Map[y][x]
				}
				gray = true
			default:
				ch = ' '
			}

			if gray != prevGray {
				if gray {
					lineBuf = append(lineBuf, []byte(csiFgGray)...)
				} else {
					lineBuf = append(lineBuf, []byte(csiReset)...)
				}
				prevGray = gray
			}
			lineBuf = append(lineBuf, ch)
		}
		if prevGray {
			lineBuf = append(lineBuf, []byte(csiReset)...)
		}
		fmt.Println(string(lineBuf))
	}

	statusBar := fmt.Sprintf(" HP:%d/%d | ATK:%d | DEF:%d | 金币:%d | 第 %d/%d 层 ",
		g.Player.HP, g.Player.MaxHP, g.Player.Atk, g.Player.Def, g.Player.Gold,
		g.Player.Floor, MaxFloor)
	padding := MapWidth - len(statusBar) - 2
	if padding < 0 {
		padding = 0
	}
	fmt.Println("═" + statusBar + "═" + string(make([]byte, padding)))
	if g.Msg != "" {
		fmt.Println(g.Msg)
	} else {
		fmt.Println(" WASD 移动  |  目标: 找到出口 >  进入下一层")
	}
}

func (g *Game) RenderBattle() {
	m := g.Battle.Monster
	fmt.Println("==================== 战斗 ====================")
	fmt.Printf(" 你        HP:%-3d/%-3d  ATK:%d  DEF:%d\n", g.Player.HP, g.Player.MaxHP, g.Player.Atk, g.Player.Def)
	fmt.Printf(" %-9s HP:%-3d/%-3d  ATK:%d  DEF:%d\n", m.Name, m.HP, m.MaxHP, m.Atk, m.Def)
	fmt.Println("----------------------------------------------")
	for _, line := range g.Battle.Log {
		if len(line) > 46 {
			line = line[:46]
		}
		fmt.Println(" " + line)
	}
	fmt.Println("==============================================")
	fmt.Println(" [A] 攻击    [F] 逃跑")
}

func (g *Game) RenderGameOver() {
	score := g.Player.Gold + g.Player.MaxFloor*50
	if g.Win {
		fmt.Println("==========================================")
		fmt.Println("           胜  利 ！                    ")
		fmt.Println("      你击败了地牢领主！                ")
		fmt.Println("==========================================")
	} else {
		fmt.Println("==========================================")
		fmt.Println("           你 死 了                      ")
		fmt.Println("==========================================")
	}
	fmt.Printf(" 最终得分： %d\n", score)
	fmt.Printf(" 到达层数： 第 %d 层 / 共 %d 层\n", g.Player.MaxFloor, MaxFloor)
	fmt.Printf(" 累计金币： %d\n", g.Player.Gold)
	fmt.Println("==========================================")
	fmt.Println(" 按 R 键重新开始，其他键退出")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
