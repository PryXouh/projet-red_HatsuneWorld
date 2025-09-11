package main

import "fmt"

func main() {
	type Personnage struct {
    Nom             string
    Classe          string
    Niveau          int
    PVMax           int
    PVActuels       int
    Potions         int
	Arme			string
	}

    hero := Personnage {
        Nom:       "Hatsune Miku",
        Classe:    "Chevalière",
        Niveau:    1,
        PVMax:     100,
        PVActuels: 50,
        Potions:   2,
		Arme:      "Sabre",
	}
	fmt.Println(hero)
}
