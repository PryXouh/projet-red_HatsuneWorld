// main.go
package main

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
	SpawnChance = 15 // % chance to spawn each tick
)

type Enemy struct {
	x, y int
}

func clearScreen() {
	fmt.Print("\x1b[2J") // clear screen
	fmt.Print("\x1b[H")  // move cursor home
}

func hideCursor() { fmt.Print("\x1b[?25l") }
func showCursor() { fmt.Print("\x1b[?25h") }

func drawBorder() {
	// top border
	for i := 0; i < Width+2; i++ {
		fmt.Print("#")
	}
	fmt.Println()
}

func drawEmptyLine() {
	fmt.Print("#")
	for i := 0; i < Width; i++ {
		fmt.Print(" ")
	}
	fmt.Println("#")
}

func drawFrame(playerX int, enemies []Enemy, score int) {
	clearScreen()
	drawBorder()
	// create a grid and place enemies/player
	grid := make([][]rune, Height)
	for y := 0; y < Height; y++ {
		grid[y] = make([]rune, Width)
		for x := 0; x < Width; x++ {
			grid[y][x] = ' '
		}
	}
	// enemies
	for _, e := range enemies {
		if e.y >= 0 && e.y < Height && e.x >= 0 && e.x < Width {
			grid[e.y][e.x] = 'X'
		}
	}
	// player (always at bottom row index Height-1)
	if playerX >= 0 && playerX < Width {
		grid[Height-1][playerX] = '@'
	}

	// print grid
	for y := 0; y < Height; y++ {
		fmt.Print("#")
		for x := 0; x < Width; x++ {
			fmt.Printf("%c", grid[y][x])
		}
		fmt.Println("#")
	}
	drawBorder()
	fmt.Printf("Score: %d    Use A/D or ←/→ to move. Press 'q' to quit.\n", score)
}

// readKeys runs in a goroutine and sends bytes to out chan
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
			// copy the bytes we received
			b := make([]byte, n)
			copy(b, buf[:n])
			out <- b
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Put terminal into raw mode so we get key-by-key input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println("Failed to set raw terminal:", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// handle ctrl+c to restore terminal
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

	// input channel
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

	// main loop
	for !gameOver {
		drawFrame(playerX, enemies, score)

		// non-blocking process input events available before tick
	loopInput:
		for {
			select {
			case b, ok := <-keyChan:
				if !ok {
					// input closed
					break loopInput
				}
				// handle common keys: arrows send escape sequences, also support 'a'/'d' and 'q'
				if len(b) == 1 {
					switch b[0] {
					case 'q', 'Q':
						gameOver = true
						break loopInput
					case 'a', 'A':
						if playerX > 0 {
							playerX--
						}
					case 'd', 'D':
						if playerX < Width-1 {
							playerX++
						}
					}
				} else if len(b) == 3 && b[0] == 0x1b && b[1] == '[' {
					// arrow keys: ESC [ A/B/C/D
					switch b[2] {
					case 'D': // left
						if playerX > 0 {
							playerX--
						}
					case 'C': // right
						if playerX < Width-1 {
							playerX++
						}
					}
				}
			default:
				break loopInput
			}
		}

		// wait for tick (but still process input in next iteration)
		<-ticker.C

		// spawn enemy?
		if rand.Intn(100) < SpawnChance {
			ex := rand.Intn(Width)
			enemies = append(enemies, Enemy{x: ex, y: 0})
		}

		// move enemies down
		newEnemies := enemies[:0]
		for _, e := range enemies {
			e.y++
			if e.y < Height {
				newEnemies = append(newEnemies, e)
			} else {
				// enemy left the screen -> increase score
				score++
			}
		}
		enemies = newEnemies

		// check collision with player (player at y = Height-1)
		for _, e := range enemies {
			if e.y == Height-1 && e.x == playerX {
				gameOver = true
				break
			}
		}
	}

	// final frame & message
	drawFrame(playerX, enemies, score)
	fmt.Println("\nGame Over! Final score:", score)
	showCursor()
	// restore terminal before exit
	term.Restore(int(os.Stdin.Fd()), oldState)
	// allow read goroutine to exit
	close(keyChan)
	wg.Wait()
}
