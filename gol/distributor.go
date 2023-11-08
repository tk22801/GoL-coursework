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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	fmt.Println("test")
	fmt.Println(fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight))
	filename := fmt.Sprintf("%dx%d", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	turn := 0

	//Create a 2D slice to store the world OK.
	//0 That's not enough on its own.
	//We actually have to get the
	//image in, so we can evolve it with
	//our game of life algorithm.
	//Or how do we do that with the IO
	//goroutine that we've just talked about?
	//So we need to work out the
	//file name from the parameters.
	//So say if we had two 256 by 256 coming in,
	//we can make out.
	//We could make a string and send that
	//down via the appropriate channel.
	//Yeah,
	//after we've sent the appropriate command.
	//We then get that image byte by
	//byte and store it in this 2D world.

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			val := <-c.ioInput
			//if val != 0 {
			//	fmt.Println(x, y)
			//}
			world[y][x] = val
		}
	}
	// TODO: Create a 2D slice to store the world.
	for Turn := 0; Turn < p.Turns; Turn++ {

		newWorld := make([][]byte, p.ImageHeight)
		for i := range world {
			newWorld[i] = make([]byte, p.ImageWidth)
		}

		// TODO: Execute all turns of the Game of Life.

		for x := 0; x < p.ImageWidth; x++ {

			for y := 0; y < p.ImageHeight; y++ {
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
				if y == 0 {
					yUp = p.ImageHeight - 1
				}
				if y == p.ImageHeight-1 {
					yDown = 0
				}
				if world[xBack][y] == 255 { //Horizontal
					numNeighbours += 1
				}
				//fmt.Println("Hello 4")
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

				//if p.Turns == 1 {
				//fmt.Println(x, y, newWorld[x][y])
				//}

			}
		}
		turn = Turn
		world = newWorld
	}
	turn += 1
	// TODO: Report the final state using FinalTurnCompleteEvent.
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
