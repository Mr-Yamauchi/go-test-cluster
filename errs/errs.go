package errs

import "fmt"

//
func CheckErrorPanic(err error, msg string) {
	if err != nil {
		fmt.Println(msg, err)
		panic(err)
	}
}

//
func CheckError(err error, msg string) (errs error) {
	if err != nil {
		fmt.Println(msg, err)
	}
	return err
}
