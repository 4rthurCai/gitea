package integration

import (
	"net/http"
	"testing"

	auth_model "gitea.dev/models/auth"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"
	api "gitea.dev/modules/structs"
	repo_service "gitea.dev/services/repository"
	"gitea.dev/tests"

	"github.com/stretchr/testify/require"
)

func TestForkBadgeMessage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)
	repo, err := repo_service.CreateRepository(t.Context(), user, user, repo_service.CreateRepoOptions{
		Name: "fork-badge-test", AutoInit: true, Readme: "Default", DefaultBranch: "main",
	})
	require.NoError(t, err)
	endpoint := "/api/v1/repos/" + repo.FullName() + "/actions/workflows/build.yml/badge"
	MakeRequest(t, NewRequest(t, "GET", endpoint), http.StatusUnauthorized)
	for _, query := range []string{"", "?branch=main", "?tag=v1.0.0"} {
		resp := MakeRequest(t, NewRequest(t, "GET", endpoint+query).AddTokenAuth(token), http.StatusOK)
		result := DecodeJSON(t, resp, &api.BadgeMessage{})
		require.Equal(t, "no status", result.Message)
	}
}
