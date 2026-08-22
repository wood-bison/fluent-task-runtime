package engine

import "github.com/wood-bison/fluent-task-runtime/contracts"

// Catalogue is the immutable profile catalogue exposed by the runtime. Task
// revisions will be added only after their image and harness digests are
// verified; a profile cannot silently imply that every task is runnable.
type Catalogue struct {
	profiles contracts.Profiles
	tasks    contracts.Tasks
}

func NewCatalogue() *Catalogue {
	tasks := []contracts.Task{
		{ID: "deferred", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "fluent-calculator", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-auth-015", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-cache-014", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-concurrency-012", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-cpu-bound-002", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-idempotency-013", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-memory-004", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "node-streams-003", Revision: 1, Profile: "node", Runtime: "Node.js 24", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "dotnet-cancellation-001", Revision: 1, Profile: "dotnet", Runtime: ".NET 10", Image: "fel-task-dotnet:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "pg-indexes-008", Revision: 1, Profile: "postgres", Runtime: "PostgreSQL 17", Image: "fel-task-postgres:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "pg-locks-016", Revision: 1, Profile: "postgres", Runtime: "PostgreSQL 17", Image: "fel-task-postgres:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "go-rate-limiter-001", Revision: 1, Profile: "go", Runtime: "Go 1.24", Image: "fel-task-go:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "java-rate-limiter-001", Revision: 1, Profile: "java", Runtime: "Java 21", Image: "fel-task-java:1", Status: "declared", Network: "none", HiddenTests: true},
	}
	profiles := []contracts.Profile{
		{ID: "node", DisplayName: "Node.js", Toolchain: "Node.js 22", Image: "fel-task-node:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "dotnet", DisplayName: ".NET", Toolchain: ".NET 9", Image: "fel-task-dotnet:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "postgres", DisplayName: "PostgreSQL", Toolchain: "PostgreSQL 17", Image: "fel-task-postgres:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "go", DisplayName: "Go", Toolchain: "Go 1.24", Image: "fel-task-go:1", Status: "declared", Network: "none", HiddenTests: true},
		{ID: "java", DisplayName: "Java", Toolchain: "JDK 21", Image: "fel-task-java:1", Status: "declared", Network: "none", HiddenTests: true},
	}
	for index := range profiles {
		for _, task := range tasks {
			if task.Profile == profiles[index].ID {
				profiles[index].SupportedTasks = append(profiles[index].SupportedTasks, task.ID)
			}
		}
	}
	return &Catalogue{profiles: contracts.Profiles{
		ContractVersion: contracts.ProfileContractVersion,
		Profiles:        profiles,
	}, tasks: contracts.Tasks{ContractVersion: contracts.TaskContractVersion, Tasks: tasks}}
}

func (c *Catalogue) Tasks() contracts.Tasks {
	result := contracts.Tasks{ContractVersion: c.tasks.ContractVersion, Tasks: make([]contracts.Task, len(c.tasks.Tasks))}
	copy(result.Tasks, c.tasks.Tasks)
	return result
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
