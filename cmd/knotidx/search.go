package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
)

func Search() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter text: ")
	text, _ := reader.ReadString('\n')
	fmt.Println(text)
	res, _ := gStore.Items()
	slog.Info("Search", "results", res)
}
