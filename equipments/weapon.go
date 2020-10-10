package equipments

//Weapon ...
type Weapon struct {
	Dices [][6]int
	Score int
}

//Weapons ...
var Weapons []Weapon

//InitWeapons ...
func InitWeapons() {
	//Basic Sword
	var basicSword Weapon
	basicSword.Dices = append(basicSword.Dices, [6]int{0, 0, 1, 1, 1, 2})
	basicSword.Score = 0
	Weapons = append(Weapons, basicSword)

	//Advanced Sword
	var advancedSword Weapon
	advancedSword.Dices = append(advancedSword.Dices, [6]int{0, 1, 1, 1, 2, 3})
	advancedSword.Score = 2
	Weapons = append(Weapons, advancedSword)

	//Double Daggeer
	var doubledagger Weapon
	doubledagger.Dices = append(doubledagger.Dices, [6]int{0, 0, 1, 1, 1, 2})
	doubledagger.Dices = append(doubledagger.Dices, [6]int{0, 0, 1, 1, 1, 2})
	doubledagger.Score = 3
	Weapons = append(Weapons, doubledagger)
}
