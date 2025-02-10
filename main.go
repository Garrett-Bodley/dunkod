package main

import (
	"dunkod/config"
	"dunkod/db"

	"fmt"
)

func init() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}
	if err := db.SetupDatabase(); err != nil {
		panic(err)
	}
	if err := db.RunMigrations(); err != nil {
		panic(err)
	}
	if err := db.ValidateMigrations(); err != nil {
		panic(err)
	}
	fmt.Println("The New York Knickerbockers are named after pants")
}

func main() {

}