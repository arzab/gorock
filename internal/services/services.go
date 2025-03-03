package services

// Import start
import "github.com/arzab/gorock/internal/services/asd"
import "github.com/arzab/gorock/internal/services/fas"
// Import end

// Services start

// Asd
var varAsd asd.Service

func Asd() asd.Service{return varAsd}
// Fas
var varFas fas.Service
func Fas() fas.Service{return varFas}
// Services end


