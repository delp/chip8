package chip8

import "fmt"

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
	Sp    uint8

	Memory [4096]uint8
	V      [16]uint8 //V registers (V0-VF)

	Pc uint16

	//TODO remove this?? v
	Opcode uint16
	I      uint16 //Index Register

	DelayTimer uint8
	SoundTimer uint8

	Gfx [64 * 32]uint8
	Key [16]uint8
}

/* info
//TODO
"The first CHIP-8 interpreter (on the COSMAC VIP computer) was also
located in RAM, from address 000 to 1FF. It would expect a CHIP-8
program to be loaded into memory after it, starting at address 200
(512 in decimal). Although modern interpreters are not in the same
memory space, you should do the same to be able to run the old programs;
you can just leave the initial space empty, except for the font."
*/

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

	nib1 := opcode & 0xF000
	X := opcode & 0x0F00
	Y := opcode & 0x00F0
	N := opcode & 0x000F
	NN := uint8(opcode & 0x00FF)
	NNN := opcode & 0x0FFF
	fmt.Println(X)
	fmt.Println(Y)
	fmt.Println(N)
	fmt.Println(NN)
	fmt.Println(NNN)

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

		case 0x07:

		}

	case 0x90:
		//9XY0 skip one instruction if VX != VY
		if c.V[X] != c.V[Y] {
			c.Pc += 0x02
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
	c.Opcode = 0 // Reset current opcode
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
	for i := 0; i < 80; i++ {
		c.Memory[i] = Fontset[i]
	}

	// Reset timers
	c.DelayTimer = 0
	c.SoundTimer = 0
}
