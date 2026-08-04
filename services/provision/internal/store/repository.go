package store

// Repository — каталог образов и PXE-конфиг (memory MVP).
type Repository interface {
	ListImages() []*OSImage
	GetImage(id string) (*OSImage, bool)
	CreateImage(img *OSImage) error
	DeleteImage(id string) bool
	PXEConfig() PXEConfig
	SetPXEConfig(cfg PXEConfig)
	RecordEnrollJob(j *EnrollJob)
	ListEnrollJobs() []*EnrollJob
}
