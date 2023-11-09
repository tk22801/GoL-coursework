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
	newWorld = makeWorld(workerHeight, p.ImageWidth)
	for x := 0; x < p.ImageWidth; x++ {
		for y := i * workerHeight; y+(i*workerHeight) < (i+1)*workerHeight; y++ {
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
				newWorld[x][y-(i*workerHeight)] = 255
			} else {
				newWorld[x][y-(i*workerHeight)] = 0
			}
			//changed y values, so it would write in newWorld correctly(which is only workerHeight tall)
			//fmt.Println(x, y, newWorld[x][y])
		}
	}
	out <- newWorld
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
			fmt.Println(newWorld)
			world = newWorld
		} else {
			out := make([]chan [][]byte, p.Threads)
			for i := range out {
				out[i] = make(chan [][]byte, p.Threads)
			}
			for i := 0; i < p.Threads; i++ {
				increment = i
				go worker(p, out[i], world, newWorld, workerHeight, increment)
				//print("go")
			}
			finalWorld := makeWorld(0, 0)
			for j := 0; j < p.Threads; j++ {
				section := <-out[j]
				finalWorld = append(finalWorld, section...)
			}
		}
		world = newWorld
		turn = Turn
	}
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
	fmt.Println(aliveCells)
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
