package utility

import (
	"github.com/kardianos/osext"
	"log"
)

func GetSkuExecutableFileName() string {
	skuExecutablePathAndFilename, err := osext.Executable()
	if err != nil {
		log.Fatalf("FATAL: could not find the executable path of the sku binary, error was: %v\n", err)
	}
	return skuExecutablePathAndFilename

}