package repository

import (
	"context"
	"encoding/json"
	"github.com/AyushCN/berth/internal/domain"
	"github.com/google/uuid"
)

// BuildRepository wraps the sqlc-generated queries
type BuildRepository struct {
	queries *Queries
}

func NewBuildRepository(q *Queries) *BuildRepository {
	return &BuildRepository{queries: q}
}

func (r *BuildRepository) CreateBuildPlan(ctx context.Context, p *domain.BuildPlan) error {
	args, _ := json.Marshal(p.BuildArgs)
	_, err := r.queries.CreateBuildPlan(ctx, CreateBuildPlanParams{
		RuntimeProfileID: p.RuntimeProfileID,
		BaseImage:        p.BaseImage,
		Dockerfile:       p.Dockerfile,
		BuildArgs:        args,
		InstallCommand:   pgText(p.InstallCommand),
		BuildCommand:     pgText(p.BuildCommand),
		StartCommand:     pgText(p.StartCommand),
		WorkingDir:       pgText(p.WorkingDir),
		Port:             pgInt4(p.Port),
		Confidence:       float32(p.Confidence),
		Status:           string(p.Status),
	})
	return err
}

func (r *BuildRepository) GetBuildPlan(ctx context.Context, id uuid.UUID) (*domain.BuildPlan, error) {
	row, err := r.queries.GetBuildPlan(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToBuildPlan(row), nil
}

func (r *BuildRepository) GetBuildPlanByRuntimeProfile(ctx context.Context, runtimeProfileID uuid.UUID) (*domain.BuildPlan, error) {
	row, err := r.queries.GetBuildPlanByRuntimeProfile(ctx, runtimeProfileID)
	if err != nil {
		return nil, err
	}
	return r.rowToBuildPlan(row), nil
}

func (r *BuildRepository) UpdateBuildPlan(ctx context.Context, p *domain.BuildPlan) error {
	args, _ := json.Marshal(p.BuildArgs)
	_, err := r.queries.UpdateBuildPlan(ctx, UpdateBuildPlanParams{
		ID:               p.ID,
		BaseImage:        pgText(p.BaseImage),
		Dockerfile:       pgText(p.Dockerfile),
		BuildArgs:        args,
		InstallCommand:   pgText(p.InstallCommand),
		BuildCommand:     pgText(p.BuildCommand),
		StartCommand:     pgText(p.StartCommand),
		WorkingDir:       pgText(p.WorkingDir),
		Port:             pgInt4(p.Port),
		Confidence:       pgFloat4(float32(p.Confidence)),
		Status:           pgText(string(p.Status)),
		Error:            pgText(p.Error),
	})
	return err
}

func (r *BuildRepository) DeleteBuildPlan(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteBuildPlan(ctx, id)
}

// Builds

func (r *BuildRepository) CreateBuild(ctx context.Context, b *domain.Build) error {
	_, err := r.queries.CreateBuild(ctx, CreateBuildParams{
		BuildPlanID: b.BuildPlanID,
		WorkspaceID: b.WorkspaceID,
		CommitHash:  b.CommitHash,
		Status:      string(b.Status),
	})
	return err
}

func (r *BuildRepository) GetBuild(ctx context.Context, id uuid.UUID) (*domain.Build, error) {
	row, err := r.queries.GetBuild(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToBuild(row), nil
}

func (r *BuildRepository) GetBuildsByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]*domain.Build, error) {
	rows, err := r.queries.GetBuildsByWorkspace(ctx, GetBuildsByWorkspaceParams{
		WorkspaceID: workspaceID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Build, len(rows))
	for i, row := range rows {
		result[i] = r.rowToBuild(row)
	}
	return result, nil
}

func (r *BuildRepository) GetBuildsByBuildPlan(ctx context.Context, buildPlanID uuid.UUID) ([]*domain.Build, error) {
	rows, err := r.queries.GetBuildsByBuildPlan(ctx, buildPlanID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Build, len(rows))
	for i, row := range rows {
		result[i] = r.rowToBuild(row)
	}
	return result, nil
}

func (r *BuildRepository) GetLatestBuildByWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.Build, error) {
	row, err := r.queries.GetLatestBuildByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return r.rowToBuild(row), nil
}

func (r *BuildRepository) UpdateBuild(ctx context.Context, b *domain.Build) error {
	_, err := r.queries.UpdateBuild(ctx, UpdateBuildParams{
		ID:               b.ID,
		Status:           pgText(string(b.Status)),
		ImageID:          uuidPtrToPgType(b.ImageID),
		Logs:             pgText(b.Logs),
		StartedAt:        pgTimestamptz(b.StartedAt),
		FinishedAt:       pgTimestamptz(b.FinishedAt),
		Error:            pgText(b.Error),
		CacheHit:         pgBool(b.CacheHit),
		BuildDurationMs:  pgInt8(int64(b.BuildDurationMs)),
	})
	return err
}

func (r *BuildRepository) DeleteBuild(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteBuild(ctx, id)
}

// Images

func (r *BuildRepository) CreateImage(ctx context.Context, img *domain.Image) error {
	labels, _ := json.Marshal(img.Labels)
	_, err := r.queries.CreateImage(ctx, CreateImageParams{
		BuildID:     uuidPtrToPgType(img.BuildID),
		WorkspaceID: img.WorkspaceID,
		Tag:         img.Tag,
		Digest:      pgText(img.Digest),
		SizeBytes:   pgInt8(img.SizeBytes),
		BaseImage:   pgText(img.BaseImage),
		Labels:      labels,
	})
	return err
}

func (r *BuildRepository) GetImage(ctx context.Context, id uuid.UUID) (*domain.Image, error) {
	row, err := r.queries.GetImage(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.rowToImage(row), nil
}

func (r *BuildRepository) GetImageByTag(ctx context.Context, tag string) (*domain.Image, error) {
	row, err := r.queries.GetImageByTag(ctx, tag)
	if err != nil {
		return nil, err
	}
	return r.rowToImage(row), nil
}

func (r *BuildRepository) GetImagesByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*domain.Image, error) {
	rows, err := r.queries.GetImagesByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Image, len(rows))
	for i, row := range rows {
		result[i] = r.rowToImage(row)
	}
	return result, nil
}

func (r *BuildRepository) UpdateImageLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.queries.UpdateImageLastUsed(ctx, id)
}

func (r *BuildRepository) DeleteImage(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteImage(ctx, id)
}

func (r *BuildRepository) rowToBuildPlan(row BuildPlan) *domain.BuildPlan {
	var args map[string]string
	if len(row.BuildArgs) > 0 {
		json.Unmarshal(row.BuildArgs, &args)
	}
	return &domain.BuildPlan{
		ID:               row.ID,
		RuntimeProfileID: row.RuntimeProfileID,
		BaseImage:        row.BaseImage,
		Dockerfile:       row.Dockerfile,
		BuildArgs:        args,
		InstallCommand:   row.InstallCommand.String,
		BuildCommand:     row.BuildCommand.String,
		StartCommand:     row.StartCommand.String,
		WorkingDir:       row.WorkingDir.String,
		Port:             int(row.Port.Int32),
		Confidence:       float64(row.Confidence),
		Status:           domain.BuildPlanStatus(row.Status),
		Error:            row.Error.String,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func (r *BuildRepository) rowToBuild(row Build) *domain.Build {
	return &domain.Build{
		ID:              row.ID,
		BuildPlanID:     row.BuildPlanID,
		WorkspaceID:     row.WorkspaceID,
		CommitHash:      row.CommitHash,
		Status:          domain.BuildStatus(row.Status),
		ImageID:         pgTypeToUUID(row.ImageID),
		Logs:            row.Logs.String,
		StartedAt:       pgTimestamptzToTimePtr(row.StartedAt),
		FinishedAt:      pgTimestamptzToTimePtr(row.FinishedAt),
		Error:           row.Error.String,
		CacheHit:        row.CacheHit.Bool,
		BuildDurationMs: row.BuildDurationMs.Int64,
		CreatedAt:       row.CreatedAt.Time,
	}
}

func (r *BuildRepository) rowToImage(row Image) *domain.Image {
	var labels map[string]string
	if len(row.Labels) > 0 {
		json.Unmarshal(row.Labels, &labels)
	}
	return &domain.Image{
		ID:          row.ID,
		BuildID:     pgTypeToUUID(row.BuildID),
		WorkspaceID: row.WorkspaceID,
		Tag:         row.Tag,
		Digest:      row.Digest.String,
		SizeBytes:   row.SizeBytes.Int64,
		BaseImage:   row.BaseImage.String,
		Labels:      labels,
		CreatedAt:   row.CreatedAt.Time,
		LastUsedAt:  pgTimestamptzToTimePtr(row.LastUsedAt),
	}
}