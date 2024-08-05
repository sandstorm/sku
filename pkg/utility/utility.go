package utility

import (
	"bufio"
	"fmt"
	"github.com/kardianos/osext"
	"log"
	"os"
	"strconv"
	"strings"
)

func GetSkuExecutableFileName() string {
	skuExecutablePathAndFilename, err := osext.Executable()
	if err != nil {
		log.Fatalf("FATAL: could not find the executable path of the sku binary, error was: %v\n", err)
	}
	return skuExecutablePathAndFilename

}

func GetNumberChoice() int {
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Your Choice: ")
		userInput, _ := reader.ReadString('\n')
		i, err := strconv.Atoi(strings.TrimSpace(userInput))
		if err == nil {
			return i
		}
	}
}
