package store

// DeployOps — software deploy/patch jobs (Stage 7c).
type DeployOps interface {
	CreateDeployJob(j *DeployJob)
	GetDeployJob(id string) (*DeployJob, bool)
	ListDeployJobs() []*DeployJob
	UpdateDeployJob(id string, status RolloutStatus) (*DeployJob, bool)
	CreatePatchJob(j *PatchJob)
	ListPatchJobs() []*PatchJob
	PlanPatches(catalog []PatchCatalogEntry) []PatchPlanRow
}

// DeployPackage — catalog entry for silent-install packages (air-gap refs).
type DeployPackage struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	PackageRef string `json:"package_ref"`
	Platform   string `json:"platform,omitempty"`
}

// PatchCatalogEntry — локальный каталог CVE→package (air-gap mirror).
type PatchCatalogEntry struct {
	CVEID      string `json:"cve_id"`
	Product    string `json:"product"`
	PackageRef string `json:"package_ref"`
}
