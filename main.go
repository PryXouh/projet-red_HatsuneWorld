package main

import "fmt"

func main() {
	type  Menu struct {
		UI1 string
		Commencer string
		Stop string
		Informations string
		UI2 string
	}

	UI := Menu {
		Commencer: "-----   Start Game: (a)",
		Stop: "Pause: (z)",
		Informations: "Menu: (m)  -----",

	
	}
	fmt.Println(UI)
}
func initCharacter() {
	type Personnage struct {
		Nom	string
		Classe string
		Niveau string
		PVMax string
		PVActuels string
		Potions string
		Armes string
		}

	hero := Personnage {
		Nom: " -----   Nom: Hatsune Miku, ",
		Classe: "Classe: Chevalière, ",
		Niveau: "Niveau: 1, ",
		PVMax: "PVMax: 100, ",
		PVActuels: "PVActuels: 50, ",
		Potions: "Potions: 2, ",
		Armes: "Armes: Sabre   ----- ",
	}
	fmt.Println(hero)
}
