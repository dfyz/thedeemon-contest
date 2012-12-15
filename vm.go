package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"strconv"
)

func Die(message string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, message, args...)
	os.Exit(1)
}

const (
	memorySize = 13371111
	startIp = 36
)

func GetOpcode(word int32) int32 {
	return word >> 16;
}

func GetOffset(word int32) int32 {
	return int32(int16(word & 0xFFFF));
}

const (
	Add = iota + 1
	Sub
	Mul
	Div
	And
	Or
	Shl
	Shr
	Not
	Jl
	Prn
)

func RunVM(memory *[memorySize]int32, ip int32, ch chan string) {
	getOperand := func(idx int32) int32 {
		return memory[ip + idx]
	}
	doBinOp := func(op func(int32, int32) int32) {
		a, b, c := getOperand(1), getOperand(2), getOperand(3)
		memory[ip + a] = op(memory[ip + b], c)
	}
	var buffer bytes.Buffer
	for {
		word := memory[ip]
		opcode, offset := GetOpcode(word), GetOffset(word)
		shouldAddOffset := true
		switch opcode {
		case Add:
			doBinOp(func(a int32, b int32) int32 { return a + b })
		case Sub:
			doBinOp(func(a int32, b int32) int32 { return a - b })
		case Mul:
			doBinOp(func(a int32, b int32) int32 { return a * b })
		case Div:
			doBinOp(func(a int32, b int32) int32 { 
				if b == 0 {
					fmt.Println("OUCH! I was about to divide by zero")
					return 0
				}
				return a / b
			})
		case And:
			doBinOp(func(a int32, b int32) int32 { return a & b })
		case Or:
			doBinOp(func(a int32, b int32) int32 { return a | b })
		case Shl:
			doBinOp(func(a int32, b int32) int32 { return a << uint(b) })
		case Shr:
			doBinOp(func(a int32, b int32) int32 { return a >> uint(b) })
		case Not:
			a, b := getOperand(1), getOperand(2)
			memory[ip + a] = ^b
		case Jl:
			a, b, c := getOperand(1), getOperand(2), getOperand(3)
			if b < c {
				ip += a
				shouldAddOffset = false
			}
		case Prn:
			a := getOperand(1)
			if a == 10 {
				ch <- buffer.String()
				buffer.Reset()
			} else {
				buffer.WriteByte(byte(a))
			}
		default:
			fmt.Printf("Found unknown opcode %d; exiting\n", opcode)
			if buffer.Len() > 0 {
				ch <- buffer.String()
				buffer.Reset()
			}
			close(ch)
			return	
		}
		if shouldAddOffset {
			ip += offset
		}
	}
}

func main() {
	f, err := os.Open("pic.bmp")
	if err != nil {
		Die("Couldn't open the file with VM data: %s\n", err)
	}

	rdr := bufio.NewReader(f)
	for {
		b, err := rdr.ReadByte()
		if err != nil {
			Die("Something went wrong while reading VM data: %s\n", err)
		}
		if b == '#' {
			rdr.UnreadByte()
			break
		}
	}

	var memory [memorySize]int32
	for i := 0; i < len(memory); i++ {
		err = binary.Read(rdr, binary.LittleEndian, &memory[i])
		if err != nil {
			if err == io.EOF {
				break
			}
			Die("Coudn't read the VM data: %s\n", err)
		}
	}
	ch := make(chan string)
	go RunVM(&memory, startIp, ch)

	regexps := []*regexp.Regexp {
		regexp.MustCompile(`^Process (\d+):$`),
		regexp.MustCompile(`^  send Value to process (\d+),$`),
		regexp.MustCompile(`  Value <- (?:([-0-9]+) \* )?[A-Z] / 64[^-0-9]*([-0-9]+)?.`),
		regexp.MustCompile(`  \[[A-Z],[A-Z],[A-Z],[A-Z]\] <- receive\(4\),`),
		regexp.MustCompile(`  [A-Z] <- \([A-Z] \+ [A-Z] \+ [A-Z] \+ [A-Z] \+ 2\) / 4,`),
	}
	currentNumbers := make([]string, 0, 100000)
	for {
		message, ok := <-ch
		if !ok {
			fmt.Println("The VM said there were no more data to send")
			break
		}
		if len(message) == 0 {
			if len(currentNumbers) > 0 {
				fmt.Println(strings.Join(currentNumbers, "\t"))
				currentNumbers = currentNumbers[:0]
			}
			continue
		}
		wasMatched := false
		for reIdx, re := range regexps {
			if matches := re.FindStringSubmatch(message); matches != nil {
				for i := 1; i < len(matches); i++ {
					strNumber := matches[i]
					if len(strNumber) == 0 {
						// Ugly and dirty, but gets the job done.
						if reIdx == 2 {
							if i == 1 {
								strNumber = "1"
							} else if i == 2 {
								strNumber = "0"
							}
						}
					}
					_, err := strconv.Atoi(strNumber)
					if err != nil {
						Die("Non-integer in \"%s\"\n", message)
					}
					currentNumbers = append(currentNumbers, strNumber)
				}

				wasMatched = true
				break
			}
		}
		if !wasMatched {
			if strings.HasPrefix(message, "  ") {
				Die("Weird message: \"%s\"\n", message)
			} else {
				fmt.Printf("!!! %s\n", message)
			}
		}
	}
}