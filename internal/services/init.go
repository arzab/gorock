package services

// Import start
import "github.com/arzab/gorock/internal/services/fas"
// Import end

func Init(configs Configs) error {
	// Init start
	
	// Fas
	varFas = fas.NewService(configs.Fas)
	// Init end
	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	// Stop start
	
	// Stop end

	return errors
}