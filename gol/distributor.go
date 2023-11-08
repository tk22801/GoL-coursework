package gol

import "fmt"

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
	world := c.ioInput
	for i, i := range {

	}
	// TODO: Create a 2D slice to store the world.
	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}

	turn := 0

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

			if p.Turns == 1 {
				fmt.Println(x, y, newWorld[x][y])
			}

		}

	}
	p.Turns = p.Turns - 1
	world = newWorld
	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
