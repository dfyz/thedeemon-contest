package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path"

	"code.google.com/p/draw2d/draw2d"
	"code.google.com/p/go.image/bmp"
)

type DrawingState struct {
	P, Dp image.Point
	Clr int
}

func GetABC(bmp image.Image, p image.Point) (a, b, c int, ok bool) {
	x, y := p.X, p.Y

	if x < 0 || y < 0 || x >= bmp.Bounds().Max.X || y >= bmp.Bounds().Max.Y {
		ok = false
		return
	}

	c_, b_, a_, _ := bmp.At(x, y).RGBA()

	trunc := func(x uint32) int { return int(int8(x)); }
	a, b, c, ok = trunc(a_), trunc(b_), trunc(c_), true
	return
}

func Next(bmp image.Image, state *DrawingState) *DrawingState {
	a, b, c, ok := GetABC(bmp, state.P)
	if (a == 0 && b == 0 && c == 0) || !ok {
		return nil
	}

	newDp := image.Point {state.Dp.X ^ a, state.Dp.Y ^ b}
	return &DrawingState {
		state.P.Add(newDp),
		newDp,
		state.Clr ^ c,
	}
}

func Prev(bmp image.Image, state *DrawingState) *DrawingState {
	oldP := state.P.Sub(state.Dp)
	a, b, c, ok := GetABC(bmp, oldP)

	if !ok {
		return nil
	}

	return &DrawingState {
		oldP,
		image.Point {state.Dp.X ^ a, state.Dp.Y ^ b},
		state.Clr ^ c,
	}
}

var letterC [7]image.Point = [7]image.Point{
	image.Point{-2, -2},
	image.Point{-2, 0},
	image.Point{-2, 2},
	image.Point{0, 4},
	image.Point{2, 2},
	image.Point{2, 0},
	image.Point{2, -2},
}

func HasLetterC(bmp image.Image, start image.Point) (startState *DrawingState, ok bool) {
	a, b, _, ok := GetABC(bmp, start)
	if !ok {
		ok = false
		return
	}

	check := func(startIdx, delta int) bool {
		startState = &DrawingState {
			start,
			image.Point {delta*letterC[startIdx].X ^ a, delta*letterC[startIdx].Y ^ b},
			0,
		}

		currentState := startState
		for i := startIdx; i >= 0 && i < len(letterC); i += delta {
			currentState = Next(bmp, currentState)
			if currentState == nil || !currentState.Dp.Eq(letterC[i].Mul(delta)) {
				return false
			}
		}
		return true
	}

	ok = check(0, 1) || check(len(letterC) - 1, -1)
	return
}

func drawImage(bmp image.Image, state *DrawingState) image.Image {
	result := image.NewRGBA(bmp.Bounds())
	draw.Draw(result, result.Bounds(), image.Black, image.ZP, draw.Src)

	g := draw2d.NewGraphicContext(result)
	g.SetStrokeColor(color.White)
	g.SetLineWidth(1)

	const maxIterations = 1000000
	var newState *DrawingState
	for i := 0; i < maxIterations; i++ {
		newState = Next(bmp, state)

		if newState == nil {
			break
		}

		g.MoveTo(float64(state.P.X), float64(state.P.Y))
		if newState.Clr != 0 {
			g.LineTo(float64(newState.P.X), float64(newState.P.Y))
			g.Stroke()
		}

		state = newState
	}

	return result
}

func SaveResult(result image.Image, state *DrawingState) {
	outputFileName := path.Join("fidonet", fmt.Sprintf("%03d-%03d-%03d-%03d.png", state.P.X, state.P.Y, state.Dp.X, state.Dp.Y))
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		Die("Coudn't open the output for writing: %s\n", outputFileName, err)
	}

	err = png.Encode(outputFile, result)
	if err != nil {
		Die("Couldn't encode the result as PNG: %s\n", err)
	}
}

func Die(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}

func main() {
	inputFile, err := os.Open("pic.bmp")
	if err != nil {
		Die("Couldn't open the input file: %s\n", err)
	}

	bmp, err := bmp.Decode(inputFile)
	if err != nil {
		Die("Coudn't decode the input BMP image: %s\n", err)
	}

	b := bmp.Bounds()
	for x := 0; x < b.Max.X; x++ {
		for y := 0; y < b.Max.Y; y++ {
			if middleState, ok := HasLetterC(bmp, image.Point{x, y}); ok {
				startState := middleState
				prevState := middleState
				for ; prevState != nil; prevState = Prev(bmp, prevState) {
					startState = prevState
				}
				startState.Clr = 0
				result := drawImage(bmp, startState)
				SaveResult(result, startState)
			}
		}
	}
}