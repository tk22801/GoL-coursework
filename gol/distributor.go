package gol

import (
	"fmt"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func makeWorld(height, width int) [][]byte {
	world := make([][]byte, height)
	for i := range world {
		world[i] = make([]byte, width)
	}
	return world
}

func worker(p Params, c distributorChannels, out chan<- [][]byte, world [][]byte, newWorld [][]byte, workerHeight int, i int, turn int) {

	yStart := i * workerHeight
	yEnd := (i + 1) * workerHeight

	//Last Section Conditions (Increased Worker Height)

	if i == p.Threads-1 && p.Threads > 1 {
		yStart = (i) * (p.ImageHeight / p.Threads)
		yEnd = p.ImageHeight
	}

	//yLowerBound and yUpperBound (Includes adjacent section's lines)

	yUpperBound := 0
	yLowerBound := 0

	if yEnd == p.ImageHeight {
		yUpperBound = 0
	} else {
		yUpperBound = yEnd
	}

	if yStart == 0 {
		yLowerBound = p.ImageHeight - 1
	} else {
		yLowerBound = yStart - 1
	}

	//println("\nTest:", i, "Lower Bound:", yLowerBound, "Upper Bound:", yUpperBound)

	newWorld = makeWorld(workerHeight+2, p.ImageWidth)

	for x := 0; x < p.ImageWidth; x++ {
		for y := yStart; y < yEnd; y++ {
			numNeighbours := 0
			xBack := x - 1
			xForward := x + 1
			yUp := y - 1
			yDown := y + 1

			if x == 0 {
				xBack = p.ImageWidth - 1
			}
			if x == p.ImageWidth-1 {
				xForward = 0
			}

			// Next Worker

			if y == 0 {
				yUp = yLowerBound
			}
			if y == p.ImageHeight-1 {
				yDown = yUpperBound
			}

			//Calculations

			if world[y][xBack] == 255 { //Horizontal
				numNeighbours += 1
			}
			if world[y][xForward] == 255 {
				numNeighbours += 1
			}
			if world[yUp][x] == 255 { //Vertical
				numNeighbours += 1
			}
			if world[yDown][x] == 255 {
				numNeighbours += 1
			}
			if world[yDown][xBack] == 255 { //Diagonal
				numNeighbours += 1
			}
			if world[yUp][xForward] == 255 {
				numNeighbours += 1
			}
			if world[yUp][xBack] == 255 {
				numNeighbours += 1
			}
			if world[yDown][xForward] == 255 {
				numNeighbours += 1
			}

			if numNeighbours == 2 && world[y][x] == 255 || numNeighbours == 3 {
				if world[y][x] == 0 {
					c.events <- CellFlipped{turn, util.Cell{y, x}}
				}
				newWorld[y-yStart][x] = 255
			} else {
				if world[y][x] == 255 {
					c.events <- CellFlipped{turn, util.Cell{y, x}}
				}
				newWorld[y-yStart][x] = 0
			}
		}
	}

	adjustedWorld := makeWorld(workerHeight, p.ImageWidth)

	for y := yStart; y < yEnd; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			adjustedWorld[y-yStart][x] = newWorld[y-yStart][x]
		}
	}

	out <- adjustedWorld
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	keyPresses := make(chan int32)
	//Events := make(chan Event)
	Run(p, c.events, keyPresses)
	Pause := "Continue"
	turn := 0
	increment := 0
	aliveCells := []util.Cell{}
	newWorld := makeWorld(0, 0)
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	for x := 0; x < p.ImageHeight; x++ {
		for y := 0; y < p.ImageWidth; y++ {
			val := <-c.ioInput
			world[x][y] = val
			if val == 255 {
				c.events <- CellFlipped{turn, util.Cell{x, y}}
			}
		}
	}
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			AliveCount := 0
			for i := 0; i < p.ImageHeight; i++ {
				for j := 0; j < p.ImageWidth; j++ {
					if world[i][j] == 255 {
						AliveCount += 1
					}
				}
			}
			c.events <- AliveCellsCount{turn + 1, AliveCount}
		}
	}()
	go func() {
		key := <-keyPresses
		if key == 's' || key == 'q' {
			c.ioCommand <- ioOutput
			filename = fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
			c.ioFilename <- filename
			for i := 0; i < p.ImageHeight; i++ {
				for j := 0; j < p.ImageWidth; j++ {
					c.ioOutput <- world[i][j]
				}
			}
			c.events <- ImageOutputComplete{turn, filename}
		}
		if key == 'q' {
			for i := 0; i < p.ImageHeight; i++ {
				for j := 0; j < p.ImageWidth; j++ {
					if world[i][j] == 255 {
						newCell := []util.Cell{{j, i}}
						aliveCells = append(aliveCells, newCell...)
					}
				}
			}
			c.events <- FinalTurnComplete{
				CompletedTurns: turn, Alive: aliveCells,
			}
			c.ioCommand <- ioCheckIdle
			<-c.ioIdle
			c.events <- StateChange{turn, Quitting}
			close(c.events)
		}
		if key == 'p' {
			if Pause == "Continue" {
				c.events <- StateChange{turn, Executing}
				Pause = "Pause"
			} else {
				if Pause == "Pause" {
					c.events <- StateChange{turn, Paused}
					Pause = "Continue"
				}
			}
		}
	}()

	for Turn := 0; Turn < p.Turns; Turn++ {
		workerHeight := p.ImageHeight / p.Threads
		if p.Threads == 1 {
			out := make(chan [][]byte)
			go worker(p, c, out, world, newWorld, workerHeight, increment, Turn)
			newWorld = <-out
			//fmt.Println(newWorld)
			world = newWorld
		} else {
			out := make([]chan [][]byte, p.Threads)
			for i := range out {
				out[i] = make(chan [][]byte, p.Threads)
			}
			for i := 0; i < p.Threads; i++ { //Makes Workers
				increment = i

				if increment == p.Threads-1 { // The last Section should include modulus if not directly divisible
					modulus := p.ImageHeight % p.Threads
					workerHeight += modulus
				}

				go worker(p, c, out[i], world, newWorld, workerHeight, increment, Turn)

			}
			finalWorld := makeWorld(0, 0) // Rebuilds world from sections
			for j := 0; j < p.Threads; j++ {
				section := <-out[j]
				finalWorld = append(finalWorld, section...)
			}
			world = finalWorld
		}
		turn = Turn
		c.events <- TurnComplete{Turn}
	}
	ticker.Stop()

	// : Report the final state using FinalTurnCompleteEvent.
	aliveCells = []util.Cell{}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			if world[i][j] == 255 {
				newCell := []util.Cell{{j, i}}
				aliveCells = append(aliveCells, newCell...)
			}
		}
	}
	//fmt.Println(aliveCells)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn, Alive: aliveCells,
	}
	c.ioCommand <- ioOutput
	filename = fmt.Sprintf("%dx%dx%d", p.ImageWidth, p.ImageHeight, turn)
	c.ioFilename <- filename
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			c.ioOutput <- world[i][j]
		}
	}
	c.events <- ImageOutputComplete{turn, filename}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
