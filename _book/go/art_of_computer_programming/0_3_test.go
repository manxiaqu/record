package art_of_computer_programming

import (
    "testing"
    "math"
    "fmt"
)

// Quicker than math.pow(x, 4) by benchmark test.
func X4(x float64) float64 {
    return x * x * x * x
}

//Try to get answer w^4 + x^4 + y^4 = z^4
//And w <= x <= y < z < 1000000
func TestGetAnswer(t *testing.T) {

    for z := float64(3); z < 1000000; z++ {

        if int(z) % 20000 == 0 {
            fmt.Println("z is", z)
        }

        z4 := X4(z)
PO:     for y := z-1; X4(y) >= z4/3; y-- {

            y4 := X4(y)
            
            for x := y; X4(x) >= (z4 - y4)/2; x-- {

                x4 := math.Pow(x, 4)

                w := math.Sqrt(z4 - y4 - x4)
                if math.Ceil(w) != w {
                    continue
                }
                
                w = math.Sqrt(w)
                if w > x {
                    break PO
                }
                if w < x {
                    continue
                }
    
                fmt.Println("right answer: w", w, "x", x, "y", y, "z", z)
            }
        }
    }
}

func BenchmarkX4(b *testing.B) {
    x := float64(100000)
    for i := 0; i < b.N; i++ {
        x = X4(x)
    }
}
//
//func BenchmarkPow(b *testing.B) {
//    x := float64(10000)
//    for i := 0; i < b.N; i++ {
//        x = math.Pow(x, 4)
//    }
//}

func BenchmarkSqrt2(b *testing.B) {
    x := float64(10000)
    for i := 0; i < b.N; i++ {
        x = math.Sqrt(x)
        if math.Ceil(x) == x {
            //
            continue
        }
        x = math.Sqrt(x)
        if math.Ceil(x) == x {
            //
            continue
        }
    }
}

//func BenchmarkS(b *testing.B) {
//    x := float64(10000)
//
//    for i := 0; i < b.N; i++ {
//        if x == 1 {
//            //
//        }
//
//
//    }
//}
