package distro_test

import (
	"fmt"
	"math/rand"

	"github.com/zephyrtronium/robot/v2/distro"
)

func Example() {
	rand.Seed(1)

	d := distro.New([]distro.Case[string]{
		{E: "bocchi", W: 10},
		{E: "nijika", W: 6},
		{E: "ryou", W: 8},
		{E: "kita", W: 8},
	})
	for i := 0; i < 5; i++ {
		fmt.Println(d.Pick(rand.Uint32()))
	}

	// Output:
	// ryou
	// kita
	// ryou
	// nijika
	// nijika
}
