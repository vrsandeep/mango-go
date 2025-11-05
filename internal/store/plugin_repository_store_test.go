package store_test

import (
	"database/sql"
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestPluginRepositoryStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	storeInstance := store.New(db)

	t.Run("Create Repository", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://raw.githubusercontent.com/test/repo/master/repository.json",
			"Test Repository",
			"Test description",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		if repo.ID == 0 {
			t.Error("Expected repository ID to be set")
		}
		if repo.URL != "https://raw.githubusercontent.com/test/repo/master/repository.json" {
			t.Errorf("Expected URL to match, got %s", repo.URL)
		}
		if repo.Name != "Test Repository" {
			t.Errorf("Expected name 'Test Repository', got '%s'", repo.Name)
		}
	})

	t.Run("Get All Repositories", func(t *testing.T) {
		repos, err := storeInstance.GetAllRepositories()
		if err != nil {
			t.Fatalf("Failed to get repositories: %v", err)
		}

		// Should have at least the default repository from migration
		if len(repos) == 0 {
			t.Error("Expected at least one repository (default)")
		}
	})

	t.Run("Get Repository By ID", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo.json",
			"Example",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		found, err := storeInstance.GetRepositoryByID(repo.ID)
		if err != nil {
			t.Fatalf("Failed to get repository: %v", err)
		}

		if found.ID != repo.ID {
			t.Errorf("Expected ID %d, got %d", repo.ID, found.ID)
		}
	})

	t.Run("Get Repository By URL", func(t *testing.T) {
		url := "https://example.com/repo2.json"
		_, err := storeInstance.CreateRepository(url, "Repo 2", "")
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		found, err := storeInstance.GetRepositoryByURL(url)
		if err != nil {
			t.Fatalf("Failed to get repository by URL: %v", err)
		}

		if found.URL != url {
			t.Errorf("Expected URL %s, got %s", url, found.URL)
		}
	})

	t.Run("Update Repository", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo3.json",
			"Original Name",
			"Original description",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		err = storeInstance.UpdateRepository(repo.ID, "Updated Name", "Updated description")
		if err != nil {
			t.Fatalf("Failed to update repository: %v", err)
		}

		updated, err := storeInstance.GetRepositoryByID(repo.ID)
		if err != nil {
			t.Fatalf("Failed to get updated repository: %v", err)
		}

		if updated.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got '%s'", updated.Name)
		}
		if updated.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
		}
	})

	t.Run("Delete Repository", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo4.json",
			"To Delete",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		err = storeInstance.DeleteRepository(repo.ID)
		if err != nil {
			t.Fatalf("Failed to delete repository: %v", err)
		}

		_, err = storeInstance.GetRepositoryByID(repo.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows after deletion, got %v", err)
		}
	})

	t.Run("Create Installed Plugin", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo5.json",
			"Repo 5",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("test-plugin", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		installed, err := storeInstance.GetInstalledPlugin("test-plugin")
		if err != nil {
			t.Fatalf("Failed to get installed plugin: %v", err)
		}

		if installed.PluginID != "test-plugin" {
			t.Errorf("Expected plugin ID 'test-plugin', got '%s'", installed.PluginID)
		}
		if installed.InstalledVersion != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", installed.InstalledVersion)
		}
	})

	t.Run("Update Installed Plugin", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo6.json",
			"Repo 6",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("update-plugin", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		// Update to new version
		err = storeInstance.CreateOrUpdateInstalledPlugin("update-plugin", repoID, "1.1.0")
		if err != nil {
			t.Fatalf("Failed to update installed plugin: %v", err)
		}

		installed, err := storeInstance.GetInstalledPlugin("update-plugin")
		if err != nil {
			t.Fatalf("Failed to get updated plugin: %v", err)
		}

		if installed.InstalledVersion != "1.1.0" {
			t.Errorf("Expected version '1.1.0', got '%s'", installed.InstalledVersion)
		}
	})

	t.Run("Delete Installed Plugin", func(t *testing.T) {
		repoID := sql.NullInt64{Int64: 1, Valid: true}
		err := storeInstance.CreateOrUpdateInstalledPlugin("delete-plugin", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		err = storeInstance.DeleteInstalledPlugin("delete-plugin")
		if err != nil {
			t.Fatalf("Failed to delete installed plugin: %v", err)
		}

		_, err = storeInstance.GetInstalledPlugin("delete-plugin")
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows after deletion, got %v", err)
		}
	})

	t.Run("Get All Installed Plugins", func(t *testing.T) {
		repo, err := storeInstance.CreateRepository(
			"https://example.com/repo7.json",
			"Repo 7",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("plugin1", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create plugin 1: %v", err)
		}
		err = storeInstance.CreateOrUpdateInstalledPlugin("plugin2", repoID, "2.0.0")
		if err != nil {
			t.Fatalf("Failed to create plugin 2: %v", err)
		}

		installed, err := storeInstance.GetAllInstalledPlugins()
		if err != nil {
			t.Fatalf("Failed to get all installed plugins: %v", err)
		}

		if len(installed) < 2 {
			t.Errorf("Expected at least 2 installed plugins, got %d", len(installed))
		}
	})
}

