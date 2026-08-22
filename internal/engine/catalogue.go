package engine

import "github.com/wood-bison/fluent-task-runtime/contracts"

// Catalogue is the immutable profile catalogue exposed by the runtime. Task
// revisions will be added only after their image and harness digests are
// verified; a profile cannot silently imply that every task is runnable.
type Catalogue struct {
	profiles contracts.Profiles
}

func NewCatalogue() *Catalogue {
	return &Catalogue{profiles: contracts.Profiles{
		ContractVersion: contracts.ProfileContractVersion,
		Profiles: []contracts.Profile{
			{ID: "node", DisplayName: "Node.js", Toolchain: "Node.js 22", Image: "fel-runtime-node@sha256:pending", Status: "declared", Network: "none", HiddenTests: true, SupportedTasks: []string{}},
			{ID: "dotnet", DisplayName: ".NET", Toolchain: ".NET 9", Image: "fel-runtime-dotnet@sha256:pending", Status: "declared", Network: "none", HiddenTests: true, SupportedTasks: []string{}},
			{ID: "postgres", DisplayName: "PostgreSQL", Toolchain: "PostgreSQL 17", Image: "fel-runtime-postgres@sha256:pending", Status: "declared", Network: "none", HiddenTests: true, SupportedTasks: []string{}},
			{ID: "go", DisplayName: "Go", Toolchain: "Go 1.24", Image: "fel-runtime-go@sha256:pending", Status: "declared", Network: "none", HiddenTests: true, SupportedTasks: []string{}},
			{ID: "java", DisplayName: "Java", Toolchain: "JDK 21", Image: "fel-runtime-java@sha256:pending", Status: "declared", Network: "none", HiddenTests: true, SupportedTasks: []string{}},
		},
	}}
}

func (c *Catalogue) Profiles() contracts.Profiles {
	result := contracts.Profiles{ContractVersion: c.profiles.ContractVersion, Profiles: make([]contracts.Profile, len(c.profiles.Profiles))}
	copy(result.Profiles, c.profiles.Profiles)
	for index := range result.Profiles {
		tasks := result.Profiles[index].SupportedTasks
		result.Profiles[index].SupportedTasks = make([]string, len(tasks))
		copy(result.Profiles[index].SupportedTasks, tasks)
	}
	return result
}
