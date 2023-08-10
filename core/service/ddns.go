package service

type IDDNSService interface {
	String() string
	Start() error
	Stop() error
}
