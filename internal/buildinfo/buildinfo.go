package buildinfo

import (
	"fmt"
)

func Print(version, date, commit string) {
	printValue("Build version", version)
	printValue("Build date", date)
	printValue("Build commit", commit)
}

func printValue(name, value string) {
	if value == "" {
		value = "N/A"
	}

	fmt.Printf("%s: %s\n", name, value)
}
