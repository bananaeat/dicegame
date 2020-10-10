package equipments

//Armor ...
type Armor struct {
	Dices [][6]int
	Score int
}

//Armors ...
var Armors []Armor

//InitArmors ...
func InitArmors() {
	//Basic Armor
	var basicArmor Armor
	basicArmor.Dices = append(basicArmor.Dices, [6]int{0, 0, 0, 0, 1, 2})
	basicArmor.Score = 0
	Armors = append(Armors, basicArmor)

	//Advanced Armor
	var advancedArmor Armor
	advancedArmor.Dices = append(advancedArmor.Dices, [6]int{0, 0, 1, 1, 1, 3})
	advancedArmor.Score = 2
	Armors = append(Armors, advancedArmor)
}
