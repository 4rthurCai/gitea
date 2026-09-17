// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package access

import (
	"strconv"
	"testing"

	actions_model "gitea.dev/models/actions"
	"gitea.dev/models/db"
	perm_model "gitea.dev/models/perm"
	repo_model "gitea.dev/models/repo"
	"gitea.dev/models/unit"
	"gitea.dev/models/unittest"
	user_model "gitea.dev/models/user"

	"github.com/stretchr/testify/require"
)

func TestForkActionsSharedRepositoryPermission(t *testing.T) {
	for _, ownerName := range []string{"actions", "AcTiOnS", "other-actions"} {
		for _, forkPR := range []bool{false, true} {
			t.Run(ownerName+"/fork="+strconv.FormatBool(forkPR), func(t *testing.T) {
				require.NoError(t, unittest.PrepareTestDatabase())
				ctx := t.Context()
				target := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
				require.True(t, target.IsPrivate)
				owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: target.OwnerID})
				_, err := db.GetEngine(ctx).ID(owner.ID).Cols("name").Update(&user_model.User{Name: ownerName})
				require.NoError(t, err)
				task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
				require.NotEqual(t, target.ID, task.RepoID)
				task.IsForkPullRequest = forkPR
				require.NoError(t, actions_model.UpdateTask(ctx, task, "is_fork_pull_request"))

				p, err := GetDoerRepoPermission(ctx, target, user_model.NewActionsUserWithTaskID(task.ID))
				require.NoError(t, err)
				require.Equal(t, ownerName != "other-actions", p.CanRead(unit.TypeCode))
				require.False(t, p.CanWrite(unit.TypeCode))
				require.False(t, p.IsAdmin())
				for _, u := range target.Units {
					require.LessOrEqual(t, p.UnitAccessMode(u.Type), perm_model.AccessModeRead)
				}

				_, err = GetActionsUserRepoPermission(ctx, target, user_model.NewActionsUser(), -1)
				require.Error(t, err)
				_, err = GetActionsUserRepoPermission(ctx, target, owner, task.ID)
				require.Error(t, err)
			})
		}
	}
}
