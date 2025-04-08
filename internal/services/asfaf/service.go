package asfaf

type service struct {
	configs Configs
}

func NewService(configs Configs) Service {
	return &service{
		configs: configs,
	}
}
