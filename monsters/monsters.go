package monsters

type mDice struct {
	Type string `json:"type"`
	Dice [6]int `json:"dice"`
}

//Monster ...
type Monster struct {
	Name  string  `json:"name"`
	HP    int     `json:"hp"`
	Score int     `json:"score"`
	Dices []mDice `json:"dices"`
}

//Monsters ...
var Monsters []Monster

//InitMonsters ...
func InitMonsters() {
	//Slime
	var slime = Monster{
		"史莱姆",
		3,
		1,
		[]mDice{
			{"ATK", [6]int{0, 0, 0, 0, 1, 2}},
		},
	}
	Monsters = append(Monsters, slime)

	//EarthSpirit
	var earthspirit = Monster{
		"大地元素",
		4,
		3,
		[]mDice{
			{"ATK", [6]int{0, 0, 0, 1, 1, 2}},
			{"DEF", [6]int{0, 0, 0, 0, 1, 1}},
		},
	}
	Monsters = append(Monsters, earthspirit)
}
