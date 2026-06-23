package main

import (
	"fmt"
	"math/rand"
)

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
