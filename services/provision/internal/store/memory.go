package store

import (
	"fmt"
	"sync"
	"time"
)

type memoryStore struct {
	mu     sync.RWMutex
	images []*OSImage
	pxe    PXEConfig
	jobs   []*EnrollJob
}

func NewMemory() Repository {
	now := time.Now().UTC()
	images := []*OSImage{
		{
			ID: "img-linux-22", Name: "Ubuntu 22.04 LTS", Platform: "linux",
			Version: "22.04", MinIORef: "s3://era-provision/images/ubuntu-22.04.iso",
			Unattended: "preseed", CreatedAt: now,
		},
		{
			ID: "img-win-2022", Name: "Windows Server 2022", Platform: "windows",
			Version: "2022", MinIORef: "s3://era-provision/images/win2022.wim",
			Unattended: "autounattend", CreatedAt: now,
		},
	}
	return &memoryStore{
		images: images,
		pxe: PXEConfig{
			TFTPRoot:     "/var/lib/era-provision/tftp",
			DefaultImage: "img-linux-22",
			BootMenu: []PXEBootEntry{
				{Label: "Ubuntu 22.04", ImageID: "img-linux-22", Kernel: "vmlinuz", Initrd: "initrd.img"},
				{Label: "Windows 2022", ImageID: "img-win-2022", Kernel: "bootmgfw.efi"},
			},
		},
	}
}

func (m *memoryStore) ListImages() []*OSImage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*OSImage, len(m.images))
	copy(out, m.images)
	return out
}

func (m *memoryStore) GetImage(id string) (*OSImage, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, img := range m.images {
		if img.ID == id {
			return img, true
		}
	}
	return nil, false
}

func (m *memoryStore) CreateImage(img *OSImage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.images {
		if existing.ID == img.ID {
			return fmt.Errorf("image %s already exists", img.ID)
		}
	}
	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	}
	m.images = append(m.images, img)
	return nil
}

func (m *memoryStore) DeleteImage(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, img := range m.images {
		if img.ID == id {
			m.images = append(m.images[:i], m.images[i+1:]...)
			return true
		}
	}
	return false
}

func (m *memoryStore) PXEConfig() PXEConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pxe
}

func (m *memoryStore) SetPXEConfig(cfg PXEConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pxe = cfg
}

func (m *memoryStore) RecordEnrollJob(j *EnrollJob) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now().UTC()
	}
	m.jobs = append(m.jobs, j)
}

func (m *memoryStore) ListEnrollJobs() []*EnrollJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*EnrollJob, len(m.jobs))
	copy(out, m.jobs)
	return out
}
