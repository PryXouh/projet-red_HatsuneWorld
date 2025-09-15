package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const Title = 
	`        .__            __                                                   .__       .___
	|  |__ _____ _/  |_  ________ __  ____   ____   __  _  _____________|  |    __| _/
	|  |  \\__  \\   __\/  ___/  |  \/    \_/ __ \  \ \/ \/ /  _ \_  __ \  |   / __ | 
	|   Y  \/ __ \|  |  \___ \|  |  /   |  \  ___/   \     (  <_> )  | \/  |__/ /_/ | 
	|___|  (______/__| /______>____/|___|__/\_____>   \/\_/ \____/|__|  |____/\_____| 
	 `

        
func main() {
	fmt.Println(Title)
	type Menu struct {
		Commencer    string
		Stop         string
		Informations string
	}

	UI := Menu{
		Commencer:    " -----   Start Game: (a)",
		Stop:         "Pause: (z)",
		Informations: "Menu: (m)  ----- ",
	}
	fmt.Println(UI)

	reader := bufio.NewReader(os.Stdin)
	counts := make(map[rune]int)

	for {
		fmt.Print("\nAppuie sur une touche : ")

		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r := []rune(line)[0]
		switch r {
		case 'a':
			fmt.Println("La partie commence !")

		case 'z':
			counts[r]++
			if counts[r]%2 == 1 {
				fmt.Println("Pause")
			} else {
				fmt.Println("La partie recommence !")
			}

		case 'm':
			counts[r]++
			if counts[r]%2 == 1 {
				fmt.Println("{ -----   Recommencer: (r), Quitter: (q)   ----- }")
			} else {
				fmt.Println("Retour au menu principal")
			}

		case 'q':
			fmt.Println("Quitter le jeu.")
			return

		case 'r':
			fmt.Println("La partie commence !")

		default:
			fmt.Printf("Touche inconnue : %q\n", r)
		}
	}
}
