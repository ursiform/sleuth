package sleuth_test

import (
	"fmt"

	"github.com/ursiform/sleuth"
)

func Example_Error() {
	config := &sleuth.Config{Interface: "bad"}
	if _, err := sleuth.New(config); err != nil {
		fmt.Printf("%v", err.(*sleuth.Error).Codes)
	}
	// Output: [905 901 900]
}
