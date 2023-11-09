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

func makeWorld(height, width int) [][]uint8 {
	world := make([][]uint8, height)
	for i := range world {
		world[i] = make([]uint8, width)
	}
	return world
}

func worker(p Params, out chan<- [][]byte, world [][]byte, workerHeight int, i int) {
	turn := 0
	newWorld := make([][]byte, 0)
	for Turn := 0; Turn < p.Turns; Turn++ {
		newWorld := make([][]byte, workerHeight)
		for i := range newWorld {
			newWorld[i] = make([]byte, p.ImageWidth)
		}
		for x := 0; x < p.ImageWidth; x++ {
			for y := 0; y+(i*p.ImageHeight) < (i+1)*p.ImageHeight; y++ {
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
					yUp = p.ImageHeight - 1
				}
				if y == p.ImageHeight-1 {
					yDown = 0
				}

				//Calculations
				if world[xBack][y] == 255 { //Horizontal
					numNeighbours += 1
				}
				if world[xForward][y] == 255 {
					numNeighbours += 1
				}
				if world[x][yUp] == 255 { //Vertical
					numNeighbours += 1
				}
				if world[x][yDown] == 255 {
					numNeighbours += 1
				}
				if world[xBack][yDown] == 255 { //Diagonal
					numNeighbours += 1
				}
				if world[xForward][yUp] == 255 {
					numNeighbours += 1
				}
				if world[xBack][yUp] == 255 {
					numNeighbours += 1
				}
				if world[xForward][yDown] == 255 {
					numNeighbours += 1
				}
				if numNeighbours == 2 && world[x][y] == 255 || numNeighbours == 3 {
					newWorld[x][y] = 255
				} else {
					newWorld[x][y] = 0
				}
			}
		}
		turn = Turn
	}
	out <- newWorld
	turn += 1
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	fmt.Println("test")
	fmt.Println(fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight))
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	turn := 0
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			world[y][x] = val
		}
	}
	workerheight := p.ImageHeight / p.Threads
	//if p.Threads == 1 {
	//out := make([]chan [][]uint8, p.Threads)
	//worker(p, out[], world, newWorld, 0)
	//newWorld = make([][]byte, 0)
	//for i := 0; i < p.Threads; i++ {
	//section := <-out[i]
	//newWorld = append(newWorld, section...)
	//}
	topLine := make([]chan [][]byte, p.Threads)
	bottomLine := make([]chan [][]byte, p.Threads)
	out := make([]chan [][]byte, p.Threads)
	for i := range out {
		out[i] = make(chan [][]byte, p.Threads)
		topLine[i] = make(chan [][]byte, p.Threads)
		bottomLine[i] = make(chan [][]byte, p.Threads)
	}

	for i := 0; i < p.Threads; i++ {
		go worker(p, out[i], world, workerheight, i)
	}
	newWorld := makeWorld(0, 0)
	for i := 0; i < p.Threads; i++ {
		section := <-out[i]
		newWorld = append(newWorld, section...)
	}
	world = newWorld
	// : Report the final state using FinalTurnCompleteEvent.
	aliveCells := []util.Cell{}
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			if world[i][j] == 255 {
				newCell := []util.Cell{{j, i}}
				aliveCells = append(aliveCells, newCell...)
			}
		}
	}

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
