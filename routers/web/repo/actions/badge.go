// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	actions_model "gitea.dev/models/actions"
	auth_model "gitea.dev/models/auth"
	"gitea.dev/modules/badge"
	"gitea.dev/modules/git"
	"gitea.dev/modules/util"
	"gitea.dev/services/context"
)

func GetWorkflowBadge(ctx *context.Context) {
	context.CheckRepoScopedToken(ctx, ctx.Repo.Repository, auth_model.Read)
	if ctx.Written() {
		return
	}

	workflowFile := ctx.PathParam("workflow_name")
	branch := ctx.FormString("branch")
	tag := ctx.FormString("tag")
	if branch == "" && tag == "" {
		branch = ctx.Repo.Repository.DefaultBranch
	}
	event := ctx.FormString("event")
	style := ctx.FormString("style")

	branchRef := git.RefNameFromBranch(branch)
	ref := branchRef.String()
	if branch == "" && tag != "" {
		ref = git.RefNameFromTag(tag).String()
	}
	b, err := getWorkflowBadge(ctx, workflowFile, ref, event)
	if err != nil {
		ctx.ServerError("GetWorkflowBadge", err)
		return
	}

	ctx.Data["Badge"] = b
	ctx.RespHeader().Set("Content-Type", "image/svg+xml")
	switch style {
	case badge.StyleFlatSquare:
		ctx.HTML(http.StatusOK, "shared/actions/runner_badge_flat-square")
	default: // defaults to badge.StyleFlat
		ctx.HTML(http.StatusOK, "shared/actions/runner_badge_flat")
	}
}

func getWorkflowBadge(ctx *context.Context, workflowFile, branchName, event string) (badge.Badge, error) {
	extension := filepath.Ext(workflowFile)
	workflowName := strings.TrimSuffix(workflowFile, extension)

	run, err := actions_model.GetWorkflowLatestRun(ctx, ctx.Repo.Repository.ID, workflowFile, branchName, event)
	if err != nil {
		if errors.Is(err, util.ErrNotExist) {
			return badge.GenerateBadge(workflowName, "no status", badge.DefaultColor), nil
		}
		return badge.Badge{}, err
	}

	color, ok := badge.GlobalVars().StatusColorMap[run.Status]
	if !ok {
		return badge.GenerateBadge(workflowName, "unknown status", badge.DefaultColor), nil
	}
	return badge.GenerateBadge(workflowName, run.Status.String(), color), nil
}
