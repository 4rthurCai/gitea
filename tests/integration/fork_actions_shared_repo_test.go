// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"gitea.dev/models/db"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	repo_service "gitea.dev/services/repository"
	"gitea.dev/tests"

	"github.com/stretchr/testify/require"
)

func TestForkActionsSharedRepositoryHTTP(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	ctx := t.Context()
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	owner.Name, owner.LowerName = "actions", "actions"
	_, err := db.GetEngine(ctx).ID(owner.ID).Cols("name", "lower_name").Update(owner)
	require.NoError(t, err)
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo, err := repo_service.CreateRepository(ctx, doer, owner, repo_service.CreateRepoOptions{
		Name: "shared-action", IsPrivate: true, AutoInit: true, Readme: "Default", DefaultBranch: "main",
	})
	require.NoError(t, err)
	// Existing fixture token belongs to a task in another repository.
	const token = "8061e833a55f6fc0157c98b883e91fcfeeb1a71a"
	apiPath := "/api/v1/repos/" + repo.FullName()
	MakeRequest(t, NewRequest(t, "GET", apiPath).AddTokenAuth(token), http.StatusOK)
	MakeRequest(t, NewRequest(t, "GET", apiPath+"/raw/README.md").AddTokenAuth(token), http.StatusOK)
	MakeRequest(t, NewRequestWithJSON(t, "PATCH", apiPath, map[string]string{"description": "not allowed"}).AddTokenAuth(token), http.StatusForbidden)
	gitPath := "/" + repo.FullName() + "/info/refs?service="
	MakeRequest(t, NewRequest(t, "GET", gitPath+"git-upload-pack").AddBasicAuth("gitea-actions", token), http.StatusOK)
	MakeRequest(t, NewRequest(t, "GET", gitPath+"git-receive-pack").AddBasicAuth("gitea-actions", token), http.StatusNotFound)
	MakeRequest(t, NewRequest(t, "GET", "/api/v1/repos/user2/repo2/raw/README.md").AddTokenAuth(token), http.StatusNotFound)
}
