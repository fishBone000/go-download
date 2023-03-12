package main

import (
	"fmt"
	"os"
)

func main() {
	var url string
	if len(os.Args) == 1 {
		fmt.Print("Speficy URL to download: ")
		for _, err := fmt.Scanf("%s", &url); err != nil; _, err = fmt.Scanf("%s", &url) {
			fmt.Printf("Invalid URL, reason %s\nSpecify URL to download: ", err.Error())
		}
	} else {
		url = os.Args[1]
	}

	res := make(chan Result)
	mng := Manager{
		MaxThread: 10,
		URL:       url,
		Res:       res,
	}
	fmt.Println("Starting download...")
	go mng.Run()
	for {
		result := <- res
		fmt.Println(result.msg)
		if result.code == Abort || result.code == Success {
			return
		}
	}
}
