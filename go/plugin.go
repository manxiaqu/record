package main

import "fmt"

var V int

func F() { fmt.Printf("Hello, number %d\n", V) }

type greeting struct {
    greet string
}

func (g greeting) GreetEnglish() {
    fmt.Println(g.greet, "in english")
}

func (g greeting) GreetChinese() {
    fmt.Println(g.greet, "in chinese")
}

func (g *greeting) SetGreet(greet string) {
    g.greet = greet
}

var Greeter greeting

//  go build -buildmode=plugin -o ./plugin.so ./plugin.go