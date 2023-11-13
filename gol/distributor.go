package gol

import (
	"fmt"
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

func worker(p Params, out chan<- [][]byte, world [][]byte, newWorld [][]byte, workerHeight int, i int) {

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
				newWorld[y-yStart][x] = 255
			} else {
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
	//fmt.Println("test")
	//fmt.Println(fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight))
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	turn := 0
	increment := 0
	newWorld := makeWorld(0, 0)
	world := makeWorld(p.ImageHeight, p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			world[y][x] = val
		}
	}
	for Turn := 0; Turn < p.Turns; Turn++ {
		workerHeight := p.ImageHeight / p.Threads
		if p.Threads == 1 {
			out := make(chan [][]byte)
			go worker(p, out, world, newWorld, workerHeight, increment)
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

				go worker(p, out[i], world, newWorld, workerHeight, increment)

			}
			finalWorld := makeWorld(0, 0) // Rebuilds world from sections
			for j := 0; j < p.Threads; j++ {
				section := <-out[j]
				finalWorld = append(finalWorld, section...)
			}
			world = finalWorld
		}
		turn = Turn
	}
	// : Report the final state using FinalTurnCompleteEvent.
	aliveCells := []util.Cell{}
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

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
