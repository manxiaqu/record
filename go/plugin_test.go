package main

import (
    "testing"
    "plugin"
)

type GreeterInterface interface {
    GreetEnglish()
    GreetChinese()
    SetGreet(greet string)
}

func TestPlugin(t *testing.T) {
    p, err := plugin.Open("plugin.so")
    if err != nil {
        panic(err)
    }
    v, err := p.Lookup("V")
    if err != nil {
        panic(err)
    }
    f, err := p.Lookup("F")
    if err != nil {
        panic(err)
    }
    *v.(*int) = 7
    f.(func())() // prints "Hello, number 7"
    
    greeter, err := p.Lookup("Greeter")
    if err != nil {
        panic(err)
    }
    
    g, ok := greeter.(GreeterInterface)
    if !ok {
        panic("")
    }
    
    g.SetGreet("first greet")
    
    g.GreetChinese()
    g.GreetEnglish()
    
    g.SetGreet("second greet word")
    
    g.GreetChinese()
    g.GreetEnglish()
}
