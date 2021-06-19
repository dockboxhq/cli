package common
import (
	"fmt"
	"strings"
)
func GetUserBoolean(prompt string, a ...interface{}) (bool, error) { 
	var input string
	for {
		fmt.Printf(prompt, a...)
		_, err := fmt.Scanf("%s", &input)
		if (err != nil) {
			return false, err
		}
		input = strings.Trim(input, "\n")
		if input == "y" || input == "Y" {
			return true, nil
		} else if input == "n" || input == "N" {
			return false, nil
		} else {
			fmt.Println("\nPlease enter 'y' or 'n'")
		}
	}
}

func GetUserString(prompt string, a ...interface{}) (string, error) {
	var input string
	fmt.Printf(prompt, a...)
	_, err := fmt.Scanf("%s", &input)
	if (err != nil) {
		return "", err
	}
	input = strings.Trim(input, "\n")
	return input, nil
}