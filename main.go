package main

import "fmt"

func main() {
	type  Menu struct {
		Commencer string
		Stop string
		Informations string
	}

	UI := Menu {
		Commencer: "Start Game: (a)",
		Stop: "Pause: (Z)",
		Informations: "Menu: (m)",
	}
	fmt.Println(UI)
}
