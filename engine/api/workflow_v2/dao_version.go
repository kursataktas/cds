package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getWorkflowVersion(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowVersion, error) {
	var dbWkfVersion dbV2WorkflowVersion
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfVersion, opts...)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbWkfVersion.V2WorkflowVersion, nil
}

func InsertWorkflowVersion(ctx context.Context, db gorpmapper.SqlExecutorWithTx, v *sdk.V2WorkflowVersion) error {
	_, next := telemetry.Span(ctx, "workflow_v2.InsertWorkflowVersion")
	defer next()
	v.ID = sdk.UUID()
	v.Created = time.Now()
	dbWkfVersion := &dbV2WorkflowVersion{V2WorkflowVersion: *v}

	if err := gorpmapping.Insert(db, dbWkfVersion); err != nil {
		return err
	}
	*v = dbWkfVersion.V2WorkflowVersion
	return nil
}

func LoadWorkflowVersion(ctx context.Context, db gorp.SqlExecutor, projKey, vcs, repository, wkf, version string) (*sdk.V2WorkflowVersion, error) {
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_version WHERE project_key = $1 AND vcs_server = $2 AND repository = $3 AND workflow_name = $4 AND version = $5").
		Args(projKey, vcs, repository, wkf, version)
	return getWorkflowVersion(ctx, db, query)
}
