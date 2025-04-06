package endpoints

type Service[HandlerFunc any] interface {
	GetPath() string
	GetMethod() string
	GetHandlers() []HandlerFunc
}
