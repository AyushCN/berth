package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AyushCN/berth/internal/domain"
	"github.com/AyushCN/berth/internal/usecase/onnx"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	ort "github.com/yalue/onnxruntime_go"
)

// ONNXInferenceEngine provides ONNX model inference capabilities
type ONNXInferenceEngine struct {
	sessions     map[string]*ort.DynamicAdvancedSession
	modelDir     string
	initialized  bool
	initMu       sync.Once
	mu           sync.RWMutex
}

// NewONNXInferenceEngine creates a new ONNX inference engine
func NewONNXInferenceEngine(modelDir string) *ONNXInferenceEngine {
	return &ONNXInferenceEngine{
		sessions: make(map[string]*ort.DynamicAdvancedSession),
		modelDir: modelDir,
	}
}

// initialize ensures the ONNX Runtime environment is initialized
func (e *ONNXInferenceEngine) initialize() error {
	var initErr error
	e.initMu.Do(func() {
		initErr = ort.InitializeEnvironment()
		if initErr == nil {
			e.initialized = true
		}
	})
	return initErr
}

// LoadModel loads an ONNX model from disk
func (e *ONNXInferenceEngine) LoadModel(ctx context.Context, modelID uuid.UUID, onnxPath string) error {
	if err := e.initialize(); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if already loaded
	if _, ok := e.sessions[modelID.String()]; ok {
		return nil
	}

	// Check file exists
	fullPath := onnxPath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(e.modelDir, onnxPath)
	}
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("ONNX model not found at %s", fullPath)
	}

	// Create session options
	sessionOpts, err := ort.NewSessionOptions()
	if err != nil {
		return fmt.Errorf("failed to create session options: %w", err)
	}
	defer sessionOpts.Destroy()

	// Create dynamic advanced session (doesn't require input/output names upfront)
	session, err := ort.NewDynamicAdvancedSession(fullPath, nil, nil, sessionOpts)
	if err != nil {
		return fmt.Errorf("failed to create ONNX session: %w", err)
	}

	e.sessions[modelID.String()] = session
	return nil
}

// UnloadModel unloads a model from memory
func (e *ONNXInferenceEngine) UnloadModel(modelID uuid.UUID) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if session, ok := e.sessions[modelID.String()]; ok {
		session.Destroy()
		delete(e.sessions, modelID.String())
	}
	return nil
}

// Predict runs inference on the loaded model
func (e *ONNXInferenceEngine) Predict(ctx context.Context, modelID uuid.UUID, features map[string]float64) (map[string]float64, error) {
	if err := e.initialize(); err != nil {
		return nil, fmt.Errorf("ONNX Runtime not initialized: %w", err)
	}

	e.mu.RLock()
	session, ok := e.sessions[modelID.String()]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("model not loaded: %s", modelID)
	}

	// For DynamicAdvancedSession, we use a default single input/output pattern
	inputNames := []string{"input"}

	// Prepare input tensor
	inputData := make([]float32, len(inputNames))

	// Map features to input order
	for i := range inputNames {
		// Use the first feature value, or 0.0
		var idx int
		for _, v := range features {
			if idx == i {
				inputData[i] = float32(v)
				break
			}
			idx++
		}
		if idx != i {
			inputData[i] = 0.0
		}
	}

	// Create input tensor
	inputTensor, err := ort.NewTensor(ort.NewShape(1, int64(len(inputNames))), inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	// Create output tensor
	outputTensor, err := ort.NewTensor(ort.NewShape(1), make([]float32, 1))
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	// Run inference
	inputs := []ort.Value{inputTensor}
	outputs := []ort.Value{outputTensor}
	err = session.Run(inputs, outputs)
	if err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	// Extract output
	result := make(map[string]float64)
	data := outputTensor.GetData()
	if len(data) > 0 {
		result["output"] = float64(data[0])
	}

	return result, nil
}

// IsLoaded checks if a model is loaded
func (e *ONNXInferenceEngine) IsLoaded(modelID uuid.UUID) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.sessions[modelID.String()]
	return ok
}

// Close closes all sessions
func (e *ONNXInferenceEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, session := range e.sessions {
		session.Destroy()
	}
	e.sessions = make(map[string]*ort.DynamicAdvancedSession)

	if e.initialized {
		ort.DestroyEnvironment()
		e.initialized = false
	}
	return nil
}

// ONNXModelExporter exports trained models to ONNX format
type ONNXModelExporter struct {
	engine *ONNXInferenceEngine
}

// NewONNXModelExporter creates a new exporter
func NewONNXModelExporter(engine *ONNXInferenceEngine) *ONNXModelExporter {
	return &ONNXModelExporter{engine: engine}
}

// ExportLinearRegression exports a linear regression model to ONNX
func (ex *ONNXModelExporter) ExportLinearRegression(ctx context.Context, model *domain.Model) (string, error) {
	weights, ok := model.Parameters["weights"].([]float64)
	if !ok {
		// Try other types
		if w, ok := model.Parameters["weights"].([]interface{}); ok {
			weights = make([]float64, len(w))
			for i, v := range w {
				if f, ok := v.(float64); ok {
					weights[i] = f
				}
			}
		} else {
			return "", fmt.Errorf("invalid weights parameter")
		}
	}

	bias, ok := model.Parameters["bias"].(float64)
	if !ok {
		if b, ok := model.Parameters["bias"].(float32); ok {
			bias = float64(b)
		} else {
			return "", fmt.Errorf("invalid bias parameter")
		}
	}

	features, ok := model.Parameters["features"].([]string)
	if !ok {
		if f, ok := model.Parameters["features"].([]interface{}); ok {
			features = make([]string, len(f))
			for i, v := range f {
				if s, ok := v.(string); ok {
					features[i] = s
				}
			}
		} else {
			return "", fmt.Errorf("invalid features parameter")
		}
	}

	// Create ONNX model file
	onnxPath, err := ex.buildLinearONNX(model.Name, model.Version, weights, bias, features)
	if err != nil {
		return "", fmt.Errorf("failed to build ONNX model: %w", err)
	}

	// Load the model into the engine
	if err := ex.engine.LoadModel(ctx, model.ID, onnxPath); err != nil {
		return "", fmt.Errorf("failed to load exported model: %w", err)
	}

	return onnxPath, nil
}

func (ex *ONNXModelExporter) buildLinearONNX(modelName, modelVersion string, weights []float64, bias float64, features []string) (string, error) {
	// Create a proper binary ONNX model using local protobuf definitions
	model := &onnx.ModelProto{
		IrVersion:      8,
		ProducerName:   "berth-prediction-engine",
		ProducerVersion: "1.0.0",
		Domain:         "ai.berth",
		ModelVersion:   1,
		DocString:      "Linear regression model for build time prediction",
		OpsetImport: []*onnx.OperatorSetIdProto{
			{Domain: "", Version: 13}, // Default ONNX opset
		},
	}

	// Create graph
	graph := &onnx.GraphProto{
		Name: "linear_regression",
		Domain: "ai.berth",
	}

	// Input tensor value info: [1, num_features]
	inputDim := []*onnx.TensorShapeProto_Dimension{
		{DimValue: 1},
		{DimValue: int64(len(features))},
	}
	inputValueInfo := &onnx.ValueInfoProto{
		Name: "input",
		Type: &onnx.TypeProto{
			Value: &onnx.TypeProto_TensorType{
				ElemType: onnx.TensorProto_FLOAT,
				Shape: &onnx.TensorShapeProto{
					Dim: inputDim,
				},
			},
		},
	}
	graph.Input = []*onnx.ValueInfoProto{inputValueInfo}

	// Output tensor value info: [1, 1]
	outputValueInfo := &onnx.ValueInfoProto{
		Name: "output",
		Type: &onnx.TypeProto{
			Value: &onnx.TypeProto_TensorType{
				ElemType: onnx.TensorProto_FLOAT,
				Shape: &onnx.TensorShapeProto{
					Dim: []*onnx.TensorShapeProto_Dimension{
						{DimValue: 1},
						{DimValue: 1},
					},
				},
			},
		},
	}
	graph.Output = []*onnx.ValueInfoProto{outputValueInfo}

	// Weight tensor (initializer)
	weightData := make([]float32, len(weights))
	for i, w := range weights {
		weightData[i] = float32(w)
	}
	weightTensor := &onnx.TensorProto{
		Name: "weights",
		DataType: int32(onnx.TensorProto_FLOAT),
		Dims: []int64{int64(len(weights)), 1},
		FloatData: weightData,
	}

	// Bias tensor (initializer)
	biasTensor := &onnx.TensorProto{
		Name: "bias",
		DataType: int32(onnx.TensorProto_FLOAT),
		Dims: []int64{1},
		FloatData: []float32{float32(bias)},
	}

	graph.Initializer = []*onnx.TensorProto{weightTensor, biasTensor}

	// MatMul node: input * weights -> matmul_out
	matmulNode := &onnx.NodeProto{
		Name: "matmul",
		OpType: "MatMul",
		Input: []string{"input", "weights"},
		Output: []string{"matmul_out"},
		Domain: "",
	}

	// Add node: matmul_out + bias -> output
	addNode := &onnx.NodeProto{
		Name: "add",
		OpType: "Add",
		Input: []string{"matmul_out", "bias"},
		Output: []string{"output"},
		Domain: "",
	}

	graph.Node = []*onnx.NodeProto{matmulNode, addNode}

	// Add opset import
	model.OpsetImport = []*onnx.OperatorSetIdProto{
		{Domain: "", Version: 13},
	}

	model.Graph = graph

	// Marshal to binary protobuf
	data, err := proto.Marshal(model)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ONNX model: %w", err)
	}

	// Write binary protobuf to file
	onnxPath := filepath.Join(ex.engine.modelDir, fmt.Sprintf("%s_%s.onnx", 
		sanitizeFileName(modelName), sanitizeFileName(modelVersion)))
	
	if err := os.WriteFile(onnxPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write ONNX file: %w", err)
	}

	return onnxPath, nil
}

func sanitizeFileName(s string) string {
	// Replace invalid filename characters
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// ONNXPredictionService wraps the prediction service with ONNX inference
type ONNXPredictionService struct {
	baseService     *PredictionService
	inferenceEngine *ONNXInferenceEngine
	exporter        *ONNXModelExporter
	mu              sync.RWMutex
	loadedModels    map[uuid.UUID]bool
}

// NewONNXPredictionService creates a new ONNX-enabled prediction service
func NewONNXPredictionService(
	baseService *PredictionService,
	modelDir string,
) *ONNXPredictionService {
	engine := NewONNXInferenceEngine(modelDir)
	return &ONNXPredictionService{
		baseService:     baseService,
		inferenceEngine: engine,
		exporter:        NewONNXModelExporter(engine),
		loadedModels:    make(map[uuid.UUID]bool),
	}
}

// PredictBuildTime runs build time prediction using ONNX if available
func (s *ONNXPredictionService) PredictBuildTime(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	return s.predict(ctx, domain.PredictionTypeBuildTime, workspaceID, features)
}

// PredictImageSize runs image size prediction using ONNX if available
func (s *ONNXPredictionService) PredictImageSize(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	return s.predict(ctx, domain.PredictionTypeImageSize, workspaceID, features)
}

// PredictCacheHit runs cache hit prediction using ONNX if available
func (s *ONNXPredictionService) PredictCacheHit(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	return s.predict(ctx, domain.PredictionTypeCacheHit, workspaceID, features)
}

// PredictFailureRisk runs failure risk prediction using ONNX if available
func (s *ONNXPredictionService) PredictFailureRisk(ctx context.Context, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	return s.predict(ctx, domain.PredictionTypeFailureRisk, workspaceID, features)
}

func (s *ONNXPredictionService) predict(ctx context.Context, pType domain.PredictionType, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	// Get best model
	model, err := s.baseService.modelTrainer.GetBestModel(ctx, pType)
	if err != nil {
		return s.fallbackPredict(ctx, pType, workspaceID, features)
	}

	// Try ONNX inference if model has ONNX path
	if model.ONNXPath != "" && s.inferenceEngine.IsLoaded(model.ID) {
		// Convert features to float64 map
		floatFeatures := make(map[string]float64)
		for k, v := range features {
			switch val := v.(type) {
			case float64:
				floatFeatures[k] = val
			case float32:
				floatFeatures[k] = float64(val)
			case int:
				floatFeatures[k] = float64(val)
			case int64:
				floatFeatures[k] = float64(val)
			case string:
				var f float64
				fmt.Sscanf(val, "%f", &f)
				floatFeatures[k] = f
			}
		}

		output, err := s.inferenceEngine.Predict(ctx, model.ID, floatFeatures)
		if err == nil && len(output) > 0 {
			// Convert output to map[string]any
			anyOutput := make(map[string]any)
			for k, v := range output {
				anyOutput[k] = v
			}

			return &domain.Prediction{
				ID:           uuid.New(),
				Type:         pType,
				WorkspaceID:  workspaceID,
				Input:        features,
				Output:       anyOutput,
				Confidence:   0.9,
				ModelVersion: model.Version,
				CreatedAt:    time.Now(),
			}, nil
		}
		// Fall back to base service on error
	}

	return s.fallbackPredict(ctx, pType, workspaceID, features)
}

func (s *ONNXPredictionService) fallbackPredict(ctx context.Context, pType domain.PredictionType, workspaceID uuid.UUID, features map[string]any) (*domain.Prediction, error) {
	switch pType {
	case domain.PredictionTypeBuildTime:
		return s.baseService.PredictBuildTime(ctx, workspaceID, features)
	case domain.PredictionTypeImageSize:
		return s.baseService.PredictImageSize(ctx, workspaceID, features)
	case domain.PredictionTypeCacheHit:
		return s.baseService.PredictCacheHit(ctx, workspaceID, features)
	case domain.PredictionTypeFailureRisk:
		return s.baseService.PredictFailureRisk(ctx, workspaceID, features)
	default:
		return s.baseService.PredictBuildTime(ctx, workspaceID, features)
	}
}

// LoadModel loads a model for ONNX inference
func (s *ONNXPredictionService) LoadModel(ctx context.Context, modelID uuid.UUID) error {
	model, err := s.baseService.modelTrainer.GetModelByID(ctx, modelID)
	if err != nil {
		return err
	}

	if model.ONNXPath == "" {
		// Export to ONNX first
		_, err = s.exporter.ExportLinearRegression(ctx, model)
		if err != nil {
			return fmt.Errorf("failed to export model: %w", err)
		}
	} else {
		// Load existing ONNX
		if err := s.inferenceEngine.LoadModel(ctx, modelID, model.ONNXPath); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.loadedModels[modelID] = true
	s.mu.Unlock()

	return nil
}

// GetLoadedModels returns IDs of loaded models
func (s *ONNXPredictionService) GetLoadedModels() []uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(s.loadedModels))
	for id := range s.loadedModels {
		ids = append(ids, id)
	}
	return ids
}

// Close closes the inference engine
func (s *ONNXPredictionService) Close() error {
	return s.inferenceEngine.Close()
}