package services

// Import start
import "github.com/arzab/gorock/internal/services/asd"
import "github.com/arzab/gorock/internal/services/fas"
// Import end


type Configs struct {
// Configs start
	
	// Asd
	Asd asd.Configs `json:"asd"`
	
	// Fas
	Fas fas.Configs `json:"fas"`
// Configs end
}

