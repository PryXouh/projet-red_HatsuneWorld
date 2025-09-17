package logic

import (
	"fmt"
	"golang.org/x/term"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	Width       = 40
	Height      = 20
	TickMs      = 120
	SpawnChance = 15
)

type Enemy struct {
	x, y int
}

// Reinitialise l'affichage du terminal avant un nouveau dessin.
func clearScreen() {
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
}

// Masque le curseur clignotant pendant la partie.
func hideCursor() { fmt.Print("\x1b[?25l") }

// Restaure le curseur du terminal quand le jeu se termine.
func showCursor() { fmt.Print("\x1b[?25h") }

// Dessine la bordure inferiere ou superieure du cadre de jeu.
func drawBorder() {
	for i := 0; i < Width+2; i++ {
		fmt.Print("#")
	}
	fmt.Println()
}

// Affiche une ligne vide du terrain entouree de murs.
func drawEmptyLine() {
	fmt.Print("#")
	for i := 0; i < Width; i++ {
		fmt.Print(" ")
	}
	fmt.Println("#")
}

// Cree une grille vide pour preparer la prochaine image.
func newEmptyGrid() [][]rune {
	grid := make([][]rune, Height)
	for y := 0; y < Height; y++ {
		row := make([]rune, Width)
		for x := range row {
			row[x] = ' '
		}
		grid[y] = row
	}
	return grid
}

// Depose chaque ennemi sur la grille si la position est valide.
func placeEnemies(grid [][]rune, enemies []Enemy) {
	for _, e := range enemies {
		if e.y >= 0 && e.y < Height && e.x >= 0 && e.x < Width {
			grid[e.y][e.x] = 'X'
		}
	}
}

// Place le joueur sur la ligne du bas.
func placePlayer(grid [][]rune, playerX int) {
	if playerX >= 0 && playerX < Width {
		grid[Height-1][playerX] = '@'
	}
}

// Affcihe la grille complete encadree de #.
func printGrid(grid [][]rune) {
	for y := 0; y < Height; y++ {
		fmt.Print("#")
		for x := 0; x < Width; x++ {
			fmt.Printf("%c", grid[y][x])
		}
		fmt.Println("#")
	}
}

// Assemble et affiche l'etat courant du jeu avec le score.
func drawFrame(playerX int, enemies []Enemy, score int) {
	clearScreen()
	drawBorder()
	grid := newEmptyGrid()
	placeEnemies(grid, enemies)
	placePlayer(grid, playerX)
	printGrid(grid)
	drawBorder()
	fmt.Printf("Score: %d    Use A/D or arrow keys to move. Press 'q' to quit.\n", score)
}

// Envoie en continu les touches pressees vers le canal d'entree.
func readKeys(out chan<- []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			close(out)
			return
		}
		if n > 0 {
			b := make([]byte, n)
			copy(b, buf[:n])
			out <- b
		}
	}
}

// Lit les entrees en attente et arrete si on doit quitter.
func consumeInputs(playerX int, keyChan <-chan []byte) (int, bool, bool) {
	currentX := playerX
	pauseRequested := false
	for {
		select {
		case b, ok := <-keyChan:
			if !ok {
				return currentX, true, pauseRequested
			}
			nextX, quit, pause := interpretKey(b, currentX)
			currentX = nextX
			if pause {
				pauseRequested = true
			}
			if quit {
				return currentX, true, pauseRequested
			}
		default:
			return currentX, false, pauseRequested
		}
	}
}

// Traduit une touche en deplacement du joueur ou en sortie.
func interpretKey(b []byte, playerX int) (int, bool, bool) {
	if len(b) == 0 {
		return playerX, false, false
	}
	if len(b) == 1 {
		switch b[0] {
		case 'q', 'Q':
			return playerX, true, false
		case 'a', 'A':
			if playerX > 0 {
				playerX--
			}
		case 'd', 'D':
			if playerX < Width-1 {
				playerX++
			}
		case 'z', 'Z', 'p', 'P':
			return playerX, false, true
		}
		return playerX, false, false
	}
	if len(b) == 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'D':
			if playerX > 0 {
				playerX--
			}
		case 'C':
			if playerX < Width-1 {
				playerX++
			}
		}
	}
	return playerX, false, false
}

// Ajoute aleatoirement un nouvel ennemi en haut de l'ecran.
func spawnEnemy(enemies []Enemy) []Enemy {
	if rand.Intn(100) < SpawnChance {
		enemies = append(enemies, Enemy{x: rand.Intn(Width), y: 0})
	}
	return enemies
}

// Fait descendre les ennemis et compte ceux qui sortent.
func advanceEnemies(enemies []Enemy) ([]Enemy, int) {
	scoreGained := 0
	next := enemies[:0]
	for _, e := range enemies {
		e.y++
		if e.y < Height {
			next = append(next, e)
		} else {
			scoreGained++
		}
	}
	return next, scoreGained
}

// Verifie si un enemi atteint la position du joueur.
func playerHit(enemies []Enemy, playerX int) bool {
	for _, e := range enemies {
		if e.y == Height-1 && e.x == playerX {
			return true
		}
	}
	return false
}

// Lance le jeu complet et attend la fin de la partie.
func RunGame() {
	rand.Seed(time.Now().UnixNano())
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Failed to set raw terminal:", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		showCursor()
		term.Restore(int(os.Stdin.Fd()), oldState)
		clearScreen()
		os.Exit(0)
	}()
	hideCursor()
	defer showCursor()
	clearScreen()
	keyChan := make(chan []byte, 10)
	var wg sync.WaitGroup
	wg.Add(1)
	go readKeys(keyChan, &wg)
	playerX := Width / 2
	enemies := make([]Enemy, 0)
	score := 0
	ticker := time.NewTicker(TickMs * time.Millisecond)
	defer ticker.Stop()
	gameOver := false
	paused := false
	for !gameOver {
		drawFrame(playerX, enemies, score)
		if paused {
			fmt.Println("\n== Pause == Appuie sur 'z' pour reprendre ou 'q' pour quitter.")
		}
		var quit bool
		var pauseToggle bool
		playerX, quit, pauseToggle = consumeInputs(playerX, keyChan)
		if pauseToggle {
			paused = !paused
		}
		if quit {
			break
		}
		<-ticker.C
		if paused {
			continue
		}
		enemies = spawnEnemy(enemies)
		var gained int
		enemies, gained = advanceEnemies(enemies)
		score += gained
		if playerHit(enemies, playerX) {
			gameOver = true
		}
	}
	drawFrame(playerX, enemies, score)
	fmt.Println("\nGame Over! Final score:", score)
	showCursor()
	term.Restore(int(os.Stdin.Fd()), oldState)
	close(keyChan)
	wg.Wait()
}
