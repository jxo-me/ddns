package service

type IDDNSService interface {
	String() string
	Hash() string
	Start() error
	Stop() error
}
