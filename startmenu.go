package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const Title = `        .__            __                                                   .__       .___
	|  |__ _____ _/  |_  ________ __  ____   ____   __  _  _____________|  |    __| _/
	|  |  \\__  \\   __\\/  ___/  |  \\    \\_/ __ \\  \\ \\/ \\/ /  _ \\_  __ \\  |   / __ | 
	|   Y  \/ __ \\  |  \\___ \\|  |  /   |  \\  ___/   \\     (  <_> )  | \\/  |__/ /_/ | 
	|___|  (______/__| /______>____/|___|__/\\_____>   \\/\\_/ \\____/|__|  |____/\\_____| `

// afficheMenu ecrit les lignes simples du menu principal.
func afficheMenu() {
	fmt.Println("-----   Start Game: (a)")
	fmt.Println("Pause pendant le jeu: (z)")
	fmt.Println("Comment jouer: (m)")
	fmt.Println("Quitter: (q)")
}

// showHelp explique les controles essentiels en langage simple.
func showHelp() {
	fmt.Println("\n--- Comment jouer a ---")
	fmt.Println("a) Choisis 'a' dans le menu pour lancer une nouvelle partie.")
	fmt.Println("b) Utilise 'a'/'d' ou les fleches gauche/droite pour bouger le personnage.")
	fmt.Println("c) Appuie sur 'z' pendant la partie pour mettre en pause, puis retape 'z' pour reprendre.")
	fmt.Println("d) Tape 'q' pendant le jeu pour revenir directement au menu.")
	fmt.Println()
}

// ShowMenu affiche le titre et attend le choix du joueur.
func ShowMenu() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println(Title)
		afficheMenu()
		fmt.Print("\nChoisis une lettre : ")

		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			continue
		}

		choice := []rune(line)[0]

		switch choice {
		case 'a', 'r':
			fmt.Println("La partie commence ! Appuie sur 'q' pour revenir ici a tout moment.")
			RunGame()
			fmt.Println("\nRetour au menu principal.\n")

		case 'z':
			fmt.Println("Pendant la partie, la touche 'z' met en pause ou relance la partie.")

		case 'm':
			showHelp()

		case 'q':
			fmt.Println("Quitter le jeu.")
			return

		default:
			fmt.Printf("Touche inconnue : %q\n", choice)
		}
	}
}
