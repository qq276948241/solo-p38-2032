package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	stdOutputHandle               = ^uintptr(10) + 1
	enableVirtualTerminalProcessing = 0x0004
)

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

func ClearScreen() {
	if f, ok := clear["windows"]; ok {
		f()
	} else if f, ok := clear["linux"]; ok {
		f()
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
		if m.Boss {
			screen[m.Y][m.X] = TileBoss
		} else {
			screen[m.Y][m.X] = TileMonster
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
