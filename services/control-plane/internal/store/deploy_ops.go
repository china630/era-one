package store

// DeployOps — software deploy/patch jobs (Stage 7c).
type DeployOps interface {
	CreateDeployJob(j *DeployJob)
	ListDeployJobs() []*DeployJob
	UpdateDeployJob(id string, status RolloutStatus) (*DeployJob, bool)
	CreatePatchJob(j *PatchJob)
	ListPatchJobs() []*PatchJob
	PlanPatches(catalog []PatchCatalogEntry) []PatchPlanRow
}

// PatchCatalogEntry — локальный каталог CVE→package (air-gap mirror).
type PatchCatalogEntry struct {
	CVEID      string `json:"cve_id"`
	Product    string `json:"product"`
	PackageRef string `json:"package_ref"`
}
