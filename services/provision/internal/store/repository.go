package store

// Repository — каталог образов и PXE-конфиг (memory MVP).
type Repository interface {
	ListImages() []*OSImage
	GetImage(id string) (*OSImage, bool)
	PXEConfig() PXEConfig
}
