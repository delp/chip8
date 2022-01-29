package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"
)

var Fontset = []uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, //0
	0x20, 0x60, 0x20, 0x20, 0x70, //1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, //2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, //3
	0x90, 0x90, 0xF0, 0x10, 0x10, //4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, //5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, //6
	0xF0, 0x10, 0x20, 0x40, 0x40, //7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, //8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, //9
	0xF0, 0x90, 0xF0, 0x90, 0x90, //A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, //B
	0xF0, 0x80, 0x80, 0x80, 0xF0, //C
	0xE0, 0x90, 0x90, 0x90, 0xE0, //D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, //E
	0xF0, 0x80, 0xF0, 0x80, 0x80, //F
}

type Chip8 struct {
	Stack [16]uint16
	Sp    uint8 //Stack Pointer

	Memory [4096]uint8
	V      [16]uint8 //V registers (V0-VF)

	Pc uint16 //Program Counter

	I uint16 //Index Register

	DelayTimer uint8
	SoundTimer uint8

	Gfx [64 * 32]uint8
	//64 pixels wide and 32 pixels tall
	Key [16]uint8 //Memory Mapped Keyboard
}

func GetCoordsFromScreenIndex(i int) (int, int) {

	y := (i / 64) + 1

	x := i - ((y - 1) * 64)
	x++

	return x, y
}

func GetScreenIndexFromCoords(x int, y int) int {
	return (x - 1) + (y-1)*64
}

func (c *Chip8) EmulateCycle() {

	// Fetch Opcode
	code1 := c.Memory[c.Pc]
	code2 := c.Memory[c.Pc+0x01]

	//build full opcode
	a := uint16(code1)
	a = a << 8
	opcode := a + uint16(code2)

	// Increment PC
	c.Pc += 0x02

	// Decode Opcode

	/*

	   "X: The second nibble. Used to look up one of the 16 registers (VX) from V0 through VF.
	    Y: The third nibble. Also used to look up one of the 16 registers (VY) from V0 through VF.
	    N: The fourth nibble. A 4-bit number.
	    NN: The second byte (third and fourth nibbles). An 8-bit immediate number.
	    NNN: The second, third and fourth nibbles. A 12-bit immediate memory address."
	*/

	//nib1
	foo := opcode & 0xF000
	foo = foo >> 8
	nib1 := uint8(foo)

	//X
	foo = opcode & 0x0F00
	foo = foo >> 8
	X := uint8(foo)

	//Y
	foo = opcode & 0x00F0
	foo = foo >> 4
	Y := uint8(foo)

	N := opcode & 0x000F
	NN := uint8(opcode & 0x00FF)
	NNN := opcode & 0x0FFF

	switch nib1 {
	case 0x00:
		if NN == 0xE0 {
			//0x00E0 clear screen
			//turn all pixels black
			for i := 0; i < 64*32; i++ {
				c.Gfx[i] = 0
			}
		} else if NN == 0xEE {
			//0x00EE return:
			//return from subroutine??
			address := c.Pop()
			c.Pc = address
		}
	case 0x10:
		//1NNN jump: set PC to NNN
		c.Pc = NNN
	case 0x20:
		//0x2NNN call:
		//call the subroutine at mem location NNN
		//unlike jump, with call you put the current
		//pc on the stack with push
		c.Push(c.Pc)
		c.Pc = NNN

	//3XNN will skip one instruction if the value in VX is equal to NN,
	//and 4XNN will skip if they are not equal.
	case 0x30:
		//3XNN skip one instruction if vx == NN
		//0x5XY0
		vx := c.V[X]
		if vx == NN {
			c.Pc += 0x02
		}
	case 0x40:
		//4XNN skip one instruction if vx != NN
		vx := c.V[X]
		if vx != NN {
			c.Pc += 0x02
		}
	//5XY0 skips if the values in VX and VY are equal
	//while 9XY0 skips if they are not equal.
	case 0x50:
		//5XY0 skip one instruction if VX == VY
		if c.V[X] == c.V[Y] {
			c.Pc += 0x02
		}

	case 0x60:
		//6XNN set reg VX to NN
		c.V[X] = NN

	case 0x70:
		//7XNN add NN to register vx
		//Note: there's no carry here, so don't fuss with the flag register
		//even on overflow
		c.V[X] = c.V[X] + NN

	case 0x80:
		switch N {
		case 0x00:
			//8XY0 set VX = VY
			c.V[X] = c.V[Y]
		case 0x01:
			//8XY1 vx = VX OR VY
			c.V[X] = c.V[X] | c.V[Y]
		case 0x02:
			//8XY2 vx = vx and vy
			c.V[X] = c.V[X] & c.V[Y]
		case 0x03:
			//8XY3 vx = vx XOR vy
			c.V[X] = c.V[X] ^ c.V[Y]

		case 0x04:
			//8XY4 vx = vx + vy
			//NOTE there's a cary this time

			//check overflow
			overflow := false
			if (c.V[X] + c.V[Y]) > 0xFF {
				overflow = true
			}

			//store the sum
			c.V[X] = c.V[X] + c.V[Y]

			//update VF
			if overflow {
				c.V[0x0F] = 1
			}

			/*
			   "8XY5 and 8XY7: SubtractPermalink

			   These both subtract the value in one register from the other, and
			   put the result in VX. In both cases, VY is not affected.

			   8XY5 sets VX to the result of VX - VY.

			   8XY7 sets VX to the result of VY - VX.

			   This subtraction will also affect the carry flag, but note that it’s
			   opposite from what you might think. If the minuend (the first operand)
			   is larger than the subtrahend (second operand), VF will be set to 1.
			   If the subtrahend is larger, and we “underflow” the result, VF is set to 0.
			   Another way of thinking of it is that VF is set to 1 before the subtraction,
			   and then the subtraction either borrows from VF (setting it to 0) or not."
			*/

		case 0x05:
			//8XY5 sets VX = VX - VY
			overflow := false
			if c.V[X] > c.V[Y] {
				overflow = true
			}

			underflow := false
			if c.V[Y] > c.V[X] {
				underflow = true
			}

			c.V[X] = c.V[X] - c.V[Y]

			if overflow {
				c.V[0xF] = 1
			}
			if underflow {
				c.V[0xF] = 0
			}

		case 0x07:
			//8XY5 sets VX = *VY* - VX

			overflow := false
			if c.V[Y] > c.V[X] {
				overflow = true
			}

			underflow := false
			if c.V[X] > c.V[Y] {
				underflow = true
			}

			c.V[X] = c.V[Y] - c.V[X]

			if overflow {
				c.V[0xF] = 1
			}
			if underflow {
				c.V[0xF] = 0
			}

			/*
				Step by step:
				(Optional, or configurable) Set VX to the value of VY
				Shift the value of VX one bit to the right (8XY6) or left (8XYE)
				Set VF to 1 if the bit that was shifted out was 1, or 0 if it was 0
			*/
		case 0x06:
			//8XY6 shift right
			c.V[X] = c.V[Y]

			foo := c.V[X] & 0x01

			c.V[X] = c.V[X] >> 1

			if foo == 0x01 {
				c.V[0x0F] = 1
			} else {
				c.V[0x0F] = 0
			}

		case 0x0E:
			//8XYE shift left

			c.V[X] = c.V[Y]

			foo := c.V[X] & 0x80

			c.V[X] = c.V[X] << 1

			if foo == 0x80 {
				c.V[0x0F] = 1
			} else {
				c.V[0x0F] = 0
			}
		}

	case 0x90:
		//9XY0 skip one instruction if VX != VY
		if c.V[X] != c.V[Y] {
			c.Pc += 0x02
		}

	case 0xA0:
		//ANNN
		//set index register to NNN

		c.I = NNN

	case 0xB0:
		//BNNN jump with offset. ambiguous and not commonly used
		//jump to NNN plus value of V0
		destination := NNN + uint16(c.V[0])
		c.Pc = destination

	case 0xC0:
		//CXNN VX = random number & VX
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)
		random := uint8(r1.Intn(255))
		c.V[X] = c.V[X] & random

	case 0xD0:
		//TODO
		//DXYN display crap

		/*
		   Set the X coordinate to the value in VX modulo 64 (or, equivalently, VX & 63, where & is the binary AND operation)
		   Set the Y coordinate to the value in VY modulo 32 (or VY & 31)
		   Set VF to 0
		*/
		xcoord := c.V[X] & 63
		ycoord := c.V[Y] & 31
		c.V[0x0F] = 0

		//For N rows:
		for i := uint16(0x00); i < N; i++ {
			//Get the Nth byte of sprite data, counting from the memory address in the I register (I is not incremented)
			spriteByte := c.Memory[c.I+N]

			//For each of the 8 pixels/bits in this sprite row:
			//TODO  //If you reach the right edge of the screen, stop drawing this row

			// 0 0x80
			// 1 0x40
			// 2 0x20
			// 3 0x10
			// 4 0x08
			// 5 0x04
			// 6 0x02
			// 7 0x01

			bit0 := spriteByte & 0x80
			if bit0 == 1 {
				//If the current pixel in the sprite row is on and the pixel at coordinates X,Y on the screen is also on, turn off the pixel and set VF to 1
				index := GetScreenIndexFromCoords(int(xcoord+0), int(ycoord))
				if c.Gfx[index] == 1 {
					c.Gfx[index] = 0
					c.V[0x0F] = 1

					//Or if the current pixel in the sprite row is on and
					//the screen pixel is not, draw the pixel at the X and Y
					//coordinates
				} else {
					c.Gfx[index] = 1
				}

				// Increment X (VX is not incremented)

			}
			// Increment Y (VY is not incremented)
			// Stop if you reach the bottom edge of the screen

		}

	case 0xE0:
		switch N {
		case 0x01:
			//EXA1 skip one instruction if the key matching VX is not pressed at this moment
			keyVal := c.Key[c.V[X]]

			if keyVal == 0x00 {
				c.Pc += 0x02
			}

		case 0x0E:
			//EX9E skip one instruction if the key matching VX *IS* pressed at this moment
			keyVal := c.Key[c.V[X]]

			if keyVal == 0x01 {
				c.Pc += 0x02
			}

		}
	case 0xF0:
		switch N {
		case 0x03:
			//FX33 binary-coded decimal conversion

			//take the number in VX, which is any from 0 - 255 (0x00 - 0xFF)
			//convert like this
			//if VX = 156 (0x9C) put 1 at c.I, 5 at c.I+1, and 6 at c.I+2

			//ok augh bear with me here....
			number := c.V[X]

			one := number / 100
			number = number - one*100

			two := number / 10
			number = number - two*10

			three := number

			c.Memory[c.I] = one
			c.Memory[c.I+0x01] = two
			c.Memory[c.I+0x01] = three
			//right?

		case 0x05:
			switch Y {
			case 0x01:
				//FX15 set delay timer to value of VX
				c.DelayTimer = c.V[X]

			case 0x05:
				//FX55 store reg to memory
				index := c.I
				for i := uint8(0); i < c.V[X]; i++ {
					c.Memory[index] = c.V[i]
					index++
				}

			case 0x06:
				//FX65 load regs from memory
				index := c.I
				for i := uint8(0); i < c.V[X]; i++ {
					c.V[i] = c.Memory[index]
					index++
				}

			}

		case 0x07:
			//FX07 set VX to value of delay timer
			c.V[X] = c.DelayTimer

		case 0x08:
			//FX18 sets sound timer to value of VX
			c.SoundTimer = c.V[X]

		case 0x09:
			//FX29 font char

			c.I = uint16(c.V[X])

		case 0x0A:
			//FX0A get key
			blocking := true
			key := uint8(0x00)
			for blocking {
				for i := 0; i < 16; i++ {
					val := c.Key[i]
					if val == 1 {
						key = val
						blocking = false
						break
					}
				}
			}
			c.V[X] = key

		case 0x0E:
			//FX1E Add to index, overflow flag set if result is
			//greater than 0x0FFF, which is outside normal memory range

			setFlag := false
			if (c.I + uint16(c.V[X])) > 0x0FFF {
				setFlag = true
			}

			c.I = c.I + uint16(c.V[X])

			if setFlag {
				c.V[0x0F] = 1
			}

		}
	}

	// Execute Opcode

	// Update timers

	//TODO display update is 60Hz
}

//TODO validate this w tests
func (c *Chip8) Push(a uint16) {
	if c.Sp < 0x0F {
		c.Sp++
		c.Stack[c.Sp] = a
	}
}

//TODO validate this w tests
func (c *Chip8) Pop() uint16 {
	a := c.Stack[c.Sp]
	if c.Sp > 0 {
		c.Sp--
	}
	return a
}

func (c *Chip8) Init() {
	c.Pc = 0x200 // Program counter starts at 0x200
	c.I = 0      // Reset index register
	c.Sp = 0     // Reset stack pointer

	// Clear display
	for i := 0; i < 2048; i++ {
		c.Gfx[i] = 0

	}

	// Clear stack, keys, regs
	for i := 0; i < 16; i++ {
		c.Stack[i] = 0
		c.Key[i] = 0
		c.V[i] = 0
	}

	// Clear Memory
	for i := 0; i < 4096; i++ {
		c.Memory[i] = 0
	}

	// Load fontset
	for i := 0; i < 5*16; i++ {
		c.Memory[i] = Fontset[i]
	}

	// Reset timers
	c.DelayTimer = 0
	c.SoundTimer = 0
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	//testfilename := "test_opcode.ch8"
	testfilename := "ibm.ch8"
	dat, err := ioutil.ReadFile(testfilename)
	check(err)

	var cpu Chip8
	cpu.Init()

	for i := 0; i < len(dat); i++ {
		offset := i + 0x200
		cpu.Memory[offset] = dat[i]
	}

	fmt.Println("booop")

	for true {
		cpu.EmulateCycle()
	}

	//copy to memory
}
