package main

import (
	"fmt"
	"hatsuneworld/start"
)

// main affiche un message d'accueil puis lance le menu de depart.
func main() {
	fmt.Println("Bienvenue dans Hatsune World !\n")
	start.ShowMenu()
}
