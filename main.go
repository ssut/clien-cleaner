package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	client := newClienClient()
	token, err := client.CSRFToken("")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Username: ")
	reader := bufio.NewReader(os.Stdin)
	username, _ := reader.ReadString('\n')

	fmt.Printf("Password: ")
	pass, err := gopass.GetPasswdMasked()
	if err != nil {
		panic(err)
	}

	succeed := client.Login(token, username, password)
	if !succeed {
		panic("failed to log into clien server")
	}
	log.Println("successfully log into client server")

	// articles := client.Articles()
	// log.Printf("found %d article(s) on your account", len(articles))

	comments := client.Comments()
	log.Printf("found %d comment(s) on your account", len(comments))

	for _, comment := range comments {
		log.Printf("Deleting comment ID %d", comment.CommentID)
		res := comment.Delete()
		log.Printf("%v\n", res)
	}

}
