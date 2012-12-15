package main

import "fmt"

import (
	"bufio"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"strings"
)

func Die(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}

const (
    processCount = 1 << 20
    initialValue = 0
    iterationCount = 7
	imageSize = 1024
)

type ProcessState struct {
	Neighbors []uint
	Value, M, A int
}

func RunIteration(states *[processCount]ProcessState, mailboxes *[processCount] chan int) {
	liveGoroutines := 0
	theyAreDone := make(chan bool)
	for i := 0; i < processCount; i++ {
		go func(pid int, iAmDone chan bool) {
			for _, neighbor := range states[pid].Neighbors {
				mailboxes[neighbor] <- states[pid].Value
			}

			incoming := [4]int{}
			for i, _ := range incoming {
				incoming[i] = <-mailboxes[pid]
			}

			tmpVal := 0
			for _, val := range incoming {
				tmpVal += val
			}
			tmpVal = (tmpVal + 2) / 4
			states[pid].Value = states[pid].M * tmpVal / 64 + states[pid].A

			iAmDone <- true
		}(i, theyAreDone)

		liveGoroutines++
	}

	for liveGoroutines > 0 {
		_ = <-theyAreDone
		liveGoroutines--
	}
}

func main() {
	rdr := bufio.NewReader(os.Stdin)
	states := [processCount]ProcessState{}

	for pid := 0; pid < processCount; pid++ {
		line, err := rdr.ReadString('\n')
		if err != nil {
			Die("Couldn't read the state for PID %d: %s\n", pid, err)
		}

		parseOrDie := func(s string) int {
			result, err := strconv.Atoi(s)
			if err != nil {
				Die("Non-integer: %s", s)
			}
			return result
		}

		line = line[:len(line)-1]
		tokens := strings.Split(line, "\t")
		states[pid] = ProcessState {
			make([]uint, len(tokens) - 3),
            initialValue,
			parseOrDie(tokens[len(tokens) - 2]),
			parseOrDie(tokens[len(tokens) - 1]),
		}
		for i := 0; i < len(states[pid].Neighbors); i++ {
			states[pid].Neighbors[i] = uint(parseOrDie(tokens[i + 1]))
		}
	}

	mailboxes := [processCount] chan int {}
	for i := 0; i < processCount; i++ {
		mailboxes[i] = make(chan int, 4)
	}

	for i := 0; i < iterationCount; i++ {
		fmt.Printf("Running iteration %d\n", i + 1)
		RunIteration(&states, &mailboxes)
	}

	result := image.NewGray(image.Rect(0, 0, imageSize, imageSize))
	for x := 0; x < imageSize; x++ {
		for y := 0; y < imageSize; y++ {
			result.Set(x, y, color.Gray{uint8(states[x*imageSize + y].Value)})
		}
	}

	f, err := os.Create("movie.png")
	if err != nil {
		Die("Couldn't create the file with the movie frame: %s\n", err)
	}

	if err = png.Encode(f, result); err != nil {
		Die("Couldn't save the movie frame: %s\n", err)
	}
}
