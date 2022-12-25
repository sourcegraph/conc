package conc

import "fmt"

func ExamplePanicCatcher() {
	var pc PanicCatcher
	i := 0
	pc.Try(func() { i += 1 })
	pc.Try(func() { panic("abort!") })
	pc.Try(func() { i += 1 })

	rc := pc.Recovered()

	fmt.Println(i)
	fmt.Println(rc.Value.(string))

	// Output:
	// 2
	// abort!
}
