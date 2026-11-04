// Command m65c02-spike demonstrates the disposable phase-cursor schema.
package main

import (
	"fmt"
	"m65c02/internal/spike"
)

func main() {
	cpu := spike.New(0x200)
	bus := &spike.RAM{}
	bus.Bytes[0x200] = 0xa9
	bus.Bytes[0x201] = 0x42
	if err := cpu.StepInstruction(bus); err != nil {
		panic(err)
	}
	fmt.Printf("spike: LDA #$42 => A=$%02X, cycles=%d, bus=%d\n", cpu.State.A, cpu.State.Cycles, len(bus.Events))
}
