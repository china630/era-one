package store

import "time"

type memoryStore struct {
	images []*OSImage
	pxe    PXEConfig
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
	out := make([]*OSImage, len(m.images))
	copy(out, m.images)
	return out
}

func (m *memoryStore) GetImage(id string) (*OSImage, bool) {
	for _, img := range m.images {
		if img.ID == id {
			return img, true
		}
	}
	return nil, false
}

func (m *memoryStore) PXEConfig() PXEConfig {
	return m.pxe
}
