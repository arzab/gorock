package cmd_service

var serviceFilesMap = map[string]string{
	"configs.go": `package %s

type Configs struct {
}
`,
	"interface.go": `package %s

type Service interface {
}
`,
	"service.go": `package %s

type service struct {
	configs Configs
}

func NewService(configs Configs) Service {
	return &service{
		configs: configs,
	}
}
`,
}

var mainFilesMap = map[string]string{
	"configs.go": `package services

// Import start
%v
// Import end


type Configs struct {
// Configs start
	%s
// Configs end
}
%s
`,
	"init.go": `package services

// Import start
%v
// Import end

func Init(configs Configs) error {
	// Init start
	%s
	// Init end
	return nil
}

func Stop() []error {
	errors := make([]error, 0)

	// Stop start
	%s
	// Stop end

	return errors
}`,
	"services.go": `package services

// Import start
%v
// Import end

// Services start
%s
// Services end

%s
`,
}
