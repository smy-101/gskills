package add

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func checkPathExists(localPath string) (bool, error) {
	_, err := os.Stat(localPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

var promptOverwrite = func() (bool, error) {
	fmt.Print("Target path already exists. Overwrite? [y/N]: ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		if err == io.EOF {
			return false, nil
		}
		return false, nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}
