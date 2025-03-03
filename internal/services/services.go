package services

// Import start
import "github.com/arzab/gorock/internal/services/fas"
// Import end

// Services start

// Fas
var varFas fas.Service

func Fas() fas.Service{return varFas}
// Services end


