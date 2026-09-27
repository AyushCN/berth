package grpc

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/AyushCN/berth/internal/domain"
	pb "github.com/AyushCN/berth/proto/gen/go/prediction"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PredictionServer implements the gRPC PredictionService
type PredictionServer struct {
	pb.UnimplementedPredictionServiceServer
	predictionService domain.PredictionService
}

func NewPredictionServer(predictionService domain.PredictionService) *PredictionServer {
	return &PredictionServer{
		predictionService: predictionService,
	}
}

func (s *PredictionServer) PredictBuildTime(ctx context.Context, req *pb.PredictBuildTimeRequest) (*pb.PredictionResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace_id: %v", err)
	}

	features := convertStringMap(req.Features)
	prediction, err := s.predictionService.PredictBuildTime(ctx, workspaceID, features)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prediction failed: %v", err)
	}

	return toPredictionResponse(prediction), nil
}

func (s *PredictionServer) PredictImageSize(ctx context.Context, req *pb.PredictImageSizeRequest) (*pb.PredictionResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace_id: %v", err)
	}

	features := convertStringMap(req.Features)
	prediction, err := s.predictionService.PredictImageSize(ctx, workspaceID, features)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prediction failed: %v", err)
	}

	return toPredictionResponse(prediction), nil
}

func (s *PredictionServer) PredictCacheHit(ctx context.Context, req *pb.PredictCacheHitRequest) (*pb.PredictionResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace_id: %v", err)
	}

	features := convertStringMap(req.Features)
	prediction, err := s.predictionService.PredictCacheHit(ctx, workspaceID, features)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prediction failed: %v", err)
	}

	return toPredictionResponse(prediction), nil
}

func (s *PredictionServer) PredictFailureRisk(ctx context.Context, req *pb.PredictFailureRiskRequest) (*pb.PredictionResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace_id: %v", err)
	}

	features := convertStringMap(req.Features)
	prediction, err := s.predictionService.PredictFailureRisk(ctx, workspaceID, features)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "prediction failed: %v", err)
	}

	return toPredictionResponse(prediction), nil
}

func (s *PredictionServer) GetPredictionHistory(ctx context.Context, req *pb.GetPredictionHistoryRequest) (*pb.GetPredictionHistoryResponse, error) {
	workspaceID, err := uuid.Parse(req.WorkspaceId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid workspace_id: %v", err)
	}

	pType := parsePredictionType(req.PredictionType)
	predictions, err := s.predictionService.GetPredictionHistory(ctx, workspaceID, pType, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get history: %v", err)
	}

	pbPredictions := make([]*pb.Prediction, len(predictions))
	for i, p := range predictions {
		pbPredictions[i] = toProtoPrediction(p)
	}

	return &pb.GetPredictionHistoryResponse{Predictions: pbPredictions}, nil
}

func (s *PredictionServer) GetModelMetrics(ctx context.Context, req *pb.GetModelMetricsRequest) (*pb.GetModelMetricsResponse, error) {
	pType := parsePredictionType(req.PredictionType)
	metrics, err := s.predictionService.GetModelMetrics(ctx, pType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get metrics: %v", err)
	}

	return &pb.GetModelMetricsResponse{Metrics: metrics}, nil
}

func (s *PredictionServer) RetrainModel(ctx context.Context, req *pb.RetrainModelRequest) (*pb.RetrainModelResponse, error) {
	pType := parsePredictionType(req.PredictionType)
	model, err := s.predictionService.RetrainModel(ctx, pType, req.Algorithm)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "retraining failed: %v", err)
	}

	return &pb.RetrainModelResponse{Model: toProtoModel(model)}, nil
}

func (s *PredictionServer) ExportModelONNX(ctx context.Context, req *pb.ExportModelONNXRequest) (*pb.ExportModelONNXResponse, error) {
	modelID, err := uuid.Parse(req.ModelId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid model_id: %v", err)
	}

	onnxPath, err := s.predictionService.ExportModelONNX(ctx, modelID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "export failed: %v", err)
	}

	return &pb.ExportModelONNXResponse{OnnxPath: onnxPath}, nil
}

func (s *PredictionServer) ActivateModel(ctx context.Context, req *pb.ActivateModelRequest) (*pb.ActivateModelResponse, error) {
	modelID, err := uuid.Parse(req.ModelId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid model_id: %v", err)
	}

	err = s.predictionService.ActivateModel(ctx, modelID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "activation failed: %v", err)
	}

	return &pb.ActivateModelResponse{Success: true}, nil
}

func (s *PredictionServer) ListModels(ctx context.Context, req *pb.ListModelsRequest) (*pb.ListModelsResponse, error) {
	_ = parsePredictionType(req.PredictionType)
	
	// Get models from model trainer (we need to add a List method to model trainer)
	// For now, return empty list
	return &pb.ListModelsResponse{Models: []*pb.Model{}}, nil
}

// Helper functions

func convertStringMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		// Try to parse as number
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			result[k] = f
		} else if b, err := strconv.ParseBool(v); err == nil {
			result[k] = b
		} else {
			result[k] = v
		}
	}
	return result
}

func toPredictionResponse(p *domain.Prediction) *pb.PredictionResponse {
	inputJSON, _ := json.Marshal(p.Input)
	outputJSON, _ := json.Marshal(p.Output)

	return &pb.PredictionResponse{
		PredictionId:  p.ID.String(),
		PredictionType: string(p.Type),
		Input:         map[string]string{"data": string(inputJSON)},
		Output:        map[string]string{"data": string(outputJSON)},
		Confidence:    p.Confidence,
		ModelVersion:  p.ModelVersion,
		CreatedAt:     p.CreatedAt.Unix(),
	}
}

func toProtoPrediction(p *domain.Prediction) *pb.Prediction {
	return &pb.Prediction{
		PredictionId:  p.ID.String(),
		PredictionType: string(p.Type),
		WorkspaceId:   p.WorkspaceID.String(),
		Input:         map[string]string{},
		Output:        map[string]string{},
		Confidence:    p.Confidence,
		ModelVersion:  p.ModelVersion,
		CreatedAt:     p.CreatedAt.Unix(),
	}
}

func toProtoModel(m *domain.Model) *pb.Model {
	paramsJSON, _ := json.Marshal(m.Parameters)
	
	return &pb.Model{
		ModelId:       m.ID.String(),
		Name:          m.Name,
		Version:       m.Version,
		PredictionType: string(m.Type),
		Algorithm:     m.Algorithm,
		Parameters:    map[string]string{"data": string(paramsJSON)},
		Metrics:       m.Metrics,
		OnnxPath:      m.ONNXPath,
		IsActive:      m.IsActive,
		CreatedAt:     m.CreatedAt.Unix(),
		UpdatedAt:     m.UpdatedAt.Unix(),
	}
}

func parsePredictionType(s string) domain.PredictionType {
	switch s {
	case "BUILD_TIME":
		return domain.PredictionTypeBuildTime
	case "IMAGE_SIZE":
		return domain.PredictionTypeImageSize
	case "CACHE_HIT":
		return domain.PredictionTypeCacheHit
	case "RESOURCE_USAGE":
		return domain.PredictionTypeResourceUsage
	case "FAILURE_RISK":
		return domain.PredictionTypeFailureRisk
	default:
		return domain.PredictionTypeBuildTime
	}
}