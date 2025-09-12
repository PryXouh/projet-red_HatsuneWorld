package main

import "fmt"

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
		Nom: "Nom: Hatsune Miku, ",
		Classe: "Classe: Chevalière, ",
		Niveau: "Niveau: 1, ",
		PVMax: "PVMax: 100, ",
		PVActuels: "PVActuels: 50, ",
		Potions: "Potions: 2, ",
		Armes: "Armes: Sabre",
	}
	fmt.Println(hero)
}
