"use client";
import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  Brain,
  TrendingUp,
  Activity,
  RotateCcw,
  Download,
  Play,
  AlertTriangle,
  CheckCircle,
  Loader2,
  BarChart2,
  Zap,
  Settings,
  ArrowRight,
  ChevronDown,
  ChevronUp,
  History,
  Target,
  Shield,
  Package,
  Clock,
} from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { useAuthStore } from "@/stores/auth";
import { api } from "@/lib/api";
import toast from "react-hot-toast";

type PredictionType = "BUILD_TIME" | "IMAGE_SIZE" | "CACHE_HIT" | "FAILURE_RISK";

interface Prediction {
  id: string;
  type: string;
  workspace_id: string;
  input: Record<string, any>;
  output: Record<string, any>;
  confidence: number;
  model_version: string;
  created_at: number;
}

interface ModelMetrics {
  mse: number;
  mae: number;
  rmse: number;
  r2: number;
  samples: number;
}

interface Model {
  id: string;
  name: string;
  version: string;
  type: string;
  algorithm: string;
  parameters: Record<string, any>;
  metrics: ModelMetrics;
  onnx_path: string;
  is_active: boolean;
  created_at: number;
  updated_at: number;
}

const typeConfig: Record<PredictionType, { color: string; icon: React.ReactNode; label: string }> = {
  BUILD_TIME: {
    color: "text-blue-400 bg-blue-400/10 border-blue-400/20",
    icon: <Clock className="w-5 h-5" />,
    label: "Build Time",
  },
  IMAGE_SIZE: {
    color: "text-purple-400 bg-purple-400/10 border-purple-400/20",
    icon: <Package className="w-5 h-5" />,
    label: "Image Size",
  },
  CACHE_HIT: {
    color: "text-emerald-400 bg-emerald-400/10 border-emerald-400/20",
    icon: <Zap className="w-5 h-5" />,
    label: "Cache Hit",
  },
  FAILURE_RISK: {
    color: "text-orange-400 bg-orange-400/10 border-orange-400/20",
    icon: <Shield className="w-5 h-5" />,
    label: "Failure Risk",
  },
};

const algorithmOptions = [
  { value: "linear", label: "Linear Regression" },
  { value: "random_forest", label: "Random Forest" },
  { value: "xgboost", label: "XGBoost" },
];

function MetricCard({ label, value, trend, icon }: { label: string; value: string | number; trend?: string; icon: React.ReactNode }) {
  return (
    <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5">
      <div className="flex items-center justify-between mb-3">
        <div className="w-9 h-9 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center">
          {icon}
        </div>
        {trend && (
          <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-400/10 text-emerald-400">
            {trend}
          </span>
        )}
      </div>
      <p className="text-2xl font-bold text-on-surface">{value}</p>
      <p className="text-xs text-on-surface-variant uppercase tracking-wider">{label}</p>
    </div>
  );
}

function PredictionCard({ prediction, type }: { prediction: Prediction; type: PredictionType }) {
  const cfg = typeConfig[type];
  const outputValue = prediction.output?.predicted_value ?? prediction.output?.output ?? "N/A";
  
  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5"
    >
      <div className="flex items-start justify-between gap-3 mb-4">
        <div className="w-9 h-9 rounded-lg bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center">
          {cfg.icon}
        </div>
        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold tracking-widest uppercase border ${cfg.color}`}>
          {cfg.label}
        </span>
      </div>
      
      <div className="space-y-3">
        <div className="bg-surface-container border border-outline-variant rounded-lg p-3">
          <p className="text-xs text-on-surface-variant uppercase tracking-wider mb-1">Predicted Value</p>
          <p className="text-xl font-mono font-bold text-on-surface">{typeof outputValue === 'number' ? outputValue.toFixed(2) : outputValue}</p>
        </div>
        
        <div className="flex items-center gap-4 text-sm text-on-surface-variant">
          <span className="flex items-center gap-1">
            <Target className="w-3.5 h-3.5" />
            Confidence: {(prediction.confidence * 100).toFixed(1)}%
          </span>
          <span className="flex items-center gap-1">
            <History className="w-3.5 h-3.5" />
            Model: {prediction.model_version}
          </span>
        </div>
        
        <p className="text-xs text-on-surface-variant/60">
          {formatDistanceToNow(new Date(prediction.created_at), { addSuffix: true })}
        </p>
      </div>
    </motion.div>
  );
}

function ModelCard({ model, onRetrain, onExport, onActivate, isLoading }: { model: Model; onRetrain: () => void; onExport: () => void; onActivate: () => void; isLoading: boolean }) {
  const isActive = model.is_active;
  
  return (
    <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5">
      <div className="flex items-start justify-between gap-3 mb-4">
        <div>
          <h3 className="font-bold text-on-surface">{model.name}</h3>
          <p className="text-xs text-on-surface-variant">{model.algorithm} • v{model.version}</p>
        </div>
        <div className="flex items-center gap-2">
          {isActive && (
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-400/10 text-emerald-400 text-xs font-semibold">
              <CheckCircle className="w-3 h-3" />
              Active
            </span>
          )}
          <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold ${isActive ? "bg-emerald-400/10 text-emerald-400" : "bg-gray-400/10 text-gray-400"}`}>
            {isActive ? "Active" : "Inactive"}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 mb-4">
        <MetricCard label="R² Score" value={model.metrics.r2.toFixed(3)} icon={<Target className="w-4 h-4 text-blue-400" />} />
        <MetricCard label="RMSE" value={model.metrics.rmse.toFixed(2)} icon={<Activity className="w-4 h-4 text-purple-400" />} />
        <MetricCard label="MAE" value={model.metrics.mae.toFixed(2)} icon={<TrendingUp className="w-4 h-4 text-emerald-400" />} />
        <MetricCard label="Samples" value={model.metrics.samples.toFixed(0)} icon={<BarChart2 className="w-4 h-4 text-orange-400" />} />
      </div>

      <div className="flex flex-wrap gap-2">
        {!isActive && (
          <button
            onClick={onActivate}
            disabled={isLoading}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-emerald-400/10 text-emerald-400 border border-emerald-400/20 rounded-lg text-sm font-semibold hover:bg-emerald-400/20 active:scale-95 transition-all cursor-pointer border-none disabled:opacity-50"
          >
            <Play className="w-3.5 h-3.5" />
            Activate
          </button>
        )}
        <button
          onClick={onRetrain}
          disabled={isLoading}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-container text-on-primary-fixed-variant rounded-lg text-sm font-semibold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-95 transition-all cursor-pointer border-none disabled:opacity-50"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          Retrain
        </button>
        {model.onnx_path && (
          <button
            onClick={onExport}
            disabled={isLoading}
            className="flex items-center gap-1.5 px-3 py-1.5 bg-surface-container border border-outline-variant rounded-lg text-sm font-semibold hover:bg-surface-container-high transition-all cursor-pointer disabled:opacity-50"
          >
            <Download className="w-3.5 h-3.5" />
            ONNX
          </button>
        )}
      </div>
    </div>
  );
}

export default function PredictionsDashboard() {
  const { user } = useAuthStore();
  const [activeTab, setActiveTab] = useState<"predict" | "models" | "history">("predict");
  const [predictionType, setPredictionType] = useState<PredictionType>("BUILD_TIME");
  const [features, setFeatures] = useState<Record<string, any>>({});
  const [prediction, setPrediction] = useState<Prediction | null>(null);
  const [predicting, setPredicting] = useState(false);
  const [models, setModels] = useState<Model[]>([]);
  const [metrics, setMetrics] = useState<ModelMetrics | null>(null);
  const [history, setHistory] = useState<Prediction[]>([]);
  const [loadingModels, setLoadingModels] = useState(false);
  const [loadingMetrics, setLoadingMetrics] = useState(false);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [retrainingModel, setRetrainingModel] = useState<string | null>(null);
  const [activeModel, setActiveModel] = useState<string | null>(null);

  useEffect(() => {
    if (user?.id) {
      fetchModels();
      fetchMetrics();
      fetchHistory();
    }
  }, [user?.id, predictionType]);

  const fetchModels = async () => {
    setLoadingModels(true);
    try {
      const modelMetrics = await api.predictions.getModelMetrics(predictionType);
      setMetrics(modelMetrics);
      // We don't have a list models endpoint yet, but we can show metrics
    } catch (err) {
      console.error("Failed to fetch models:", err);
    } finally {
      setLoadingModels(false);
    }
  };

  const fetchMetrics = async () => {
    setLoadingMetrics(true);
    try {
      const modelMetrics = await api.predictions.getModelMetrics(predictionType);
      setMetrics(modelMetrics);
    } catch (err) {
      console.error("Failed to fetch metrics:", err);
    } finally {
      setLoadingMetrics(false);
    }
  };

  const fetchHistory = async () => {
    setLoadingHistory(true);
    try {
      const hist = await api.predictions.getHistory(user!.id, predictionType, 20, 0);
      setHistory(hist);
    } catch (err) {
      console.error("Failed to fetch history:", err);
    } finally {
      setLoadingHistory(false);
    }
  };

  const handlePredict = async () => {
    if (!user?.id) return;
    setPredicting(true);
    try {
      let result: Prediction;
      switch (predictionType) {
        case "BUILD_TIME":
          result = await api.predictions.predictBuildTime(user.id, features);
          break;
        case "IMAGE_SIZE":
          result = await api.predictions.predictImageSize(user.id, features);
          break;
        case "CACHE_HIT":
          result = await api.predictions.predictCacheHit(user.id, features);
          break;
        case "FAILURE_RISK":
          result = await api.predictions.predictFailureRisk(user.id, features);
          break;
      }
      setPrediction(result);
      toast.success("Prediction generated!");
      fetchHistory();
    } catch (err: any) {
      toast.error(err.message || "Prediction failed");
    } finally {
      setPredicting(false);
    }
  };

  const handleRetrain = async (algorithm: string) => {
    setRetrainingModel(algorithm);
    try {
      await api.predictions.retrainModel(predictionType, algorithm);
      toast.success(`Retraining ${algorithm} started!`);
      fetchMetrics();
    } catch (err: any) {
      toast.error(err.message || "Retraining failed");
    } finally {
      setRetrainingModel(null);
    }
  };

  const handleActivate = async () => {
    // Not implemented in backend yet
    toast("Model activation coming soon");
  };

  const handleExport = async () => {
    try {
      const result = await api.predictions.exportModel(""); // Would need model ID
      toast.success("Model exported to ONNX!");
    } catch (err: any) {
      toast.error(err.message || "Export failed");
    }
  };

  const featureConfigs: Record<PredictionType, { label: string; key: string; type: "number" | "select"; options?: string[] }[]> = {
    BUILD_TIME: [
      { label: "Language", key: "language", type: "select", options: ["node", "python", "go", "rust", "java"] },
      { label: "Framework", key: "framework", type: "select", options: ["next", "express", "fastapi", "django", "gin", "axum", "spring"] },
      { label: "Port", key: "exposed_port", type: "number" },
      { label: "Lockfiles Count", key: "num_lockfiles", type: "number" },
      { label: "Has Dockerfile", key: "has_dockerfile", type: "select", options: ["true", "false"] },
    ],
    IMAGE_SIZE: [
      { label: "Language", key: "language", type: "select", options: ["node", "python", "go", "rust", "java"] },
      { label: "Framework", key: "framework", type: "select", options: ["next", "express", "fastapi", "django", "gin", "axum", "spring"] },
      { label: "Base Image", key: "base_image", type: "select", options: ["node:20-alpine", "python:3.11-slim", "golang:1.23-alpine", "rust:1.80-slim", "eclipse-temurin:21-jdk-alpine"] },
    ],
    CACHE_HIT: [
      { label: "Language", key: "language", type: "select", options: ["node", "python", "go", "rust", "java"] },
      { label: "Cache Key", key: "cache_key", type: "number" },
      { label: "Lockfiles Count", key: "num_lockfiles", type: "number" },
    ],
    FAILURE_RISK: [
      { label: "Language", key: "language", type: "select", options: ["node", "python", "go", "rust", "java"] },
      { label: "Framework", key: "framework", type: "select", options: ["next", "express", "fastapi", "django", "gin", "axum", "spring"] },
      { label: "Ambiguous Entry", key: "ambiguous_entry", type: "select", options: ["true", "false"] },
    ],
  };

  return (
    <div className="space-y-8 pb-12">
      {/* Page Header */}
      <div className="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-primary-fixed/10 border border-primary-fixed/20 flex items-center justify-center">
            <Brain className="w-5 h-5 text-primary-fixed" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-on-surface">
              Prediction Engine
            </h1>
            <p className="text-on-surface-variant text-sm">
              ML-powered predictions for build optimization
            </p>
          </div>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="flex gap-1 bg-surface-container-lowest border border-outline-variant rounded-xl p-1">
        {[
          { id: "predict", label: "Predict", icon: <Zap className="w-4 h-4" /> },
          { id: "models", label: "Models", icon: <Brain className="w-4 h-4" /> },
          { id: "history", label: "History", icon: <History className="w-4 h-4" /> },
        ].map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id as any)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
              activeTab === tab.id
                ? "bg-primary-container text-on-primary-fixed-variant shadow-[0_0_20px_rgba(0,240,255,0.2)]"
                : "text-on-surface-variant hover:bg-surface-container"
            }`}
          >
            {tab.icon}
            {tab.label}
          </button>
        ))}
      </div>

      {/* Prediction Type Selector */}
      <div className="flex flex-wrap gap-2">
        {(Object.keys(typeConfig) as PredictionType[]).map((type) => {
          const cfg = typeConfig[type];
          return (
            <button
              key={type}
              onClick={() => { setPredictionType(type); setPrediction(null); }}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl border text-sm font-semibold transition-all ${
                predictionType === type
                  ? `bg-${cfg.color.split(" ")[0].replace("text-", "bg-").replace("border-", "border-")} text-white border-transparent`
                  : "bg-surface-container-lowest border-outline-variant text-on-surface hover:bg-surface-container"
              }`}
            >
              {cfg.icon}
              {cfg.label}
            </button>
          );
        })}
      </div>

      {/* Tab Content */}
      {activeTab === "predict" && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Feature Input Panel */}
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5"
          >
            <h2 className="font-bold text-on-surface mb-5 flex items-center gap-2">
              <Settings className="w-5 h-5" />
              Input Features
            </h2>
            
            <div className="space-y-4">
              {featureConfigs[predictionType].map((config) => (
                <div key={config.key} className="space-y-1.5">
                  <label className="text-xs font-semibold text-on-surface-variant uppercase tracking-wider">
                    {config.label}
                  </label>
                  {config.type === "select" ? (
                    <select
                      value={features[config.key] || ""}
                      onChange={(e) => setFeatures({ ...features, [config.key]: e.target.value })}
                      className="w-full bg-surface-container border border-outline-variant rounded-lg px-3 py-2 text-sm text-on-surface focus:outline-none focus:ring-2 focus:ring-primary-fixed/30 focus:border-primary-fixed"
                    >
                      <option value="">Select...</option>
                      {config.options?.map((opt) => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                  ) : (
                    <input
                      type="number"
                      value={features[config.key] || ""}
                      onChange={(e) => setFeatures({ ...features, [config.key]: parseFloat(e.target.value) || 0 })}
                      placeholder={`Enter ${config.label.toLowerCase()}`}
                      className="w-full bg-surface-container border border-outline-variant rounded-lg px-3 py-2 text-sm text-on-surface focus:outline-none focus:ring-2 focus:ring-primary-fixed/30 focus:border-primary-fixed"
                    />
                  )}
                </div>
              ))}
            </div>

            <button
              onClick={handlePredict}
              disabled={predicting}
              className="w-full mt-6 flex items-center justify-center gap-2 bg-primary-container text-on-primary-fixed-variant px-6 py-3 rounded-xl font-bold hover:shadow-[0_0_20px_rgba(0,240,255,0.2)] active:scale-[0.98] transition-all cursor-pointer border-none disabled:opacity-50"
            >
              {predicting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Predicting...
                </>
              ) : (
                <>
                  <Zap className="w-4 h-4" />
                  Generate Prediction
                </>
              )}
            </button>
          </motion.div>

          {/* Prediction Result */}
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5 h-full"
          >
            <h2 className="font-bold text-on-surface mb-5 flex items-center gap-2">
              <Target className="w-5 h-5" />
              Prediction Result
            </h2>

            {prediction ? (
              <PredictionCard prediction={prediction} type={predictionType} />
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-center p-8">
                <div className="w-16 h-16 rounded-2xl bg-primary-fixed/5 border border-primary-fixed/10 flex items-center justify-center mb-5 mx-auto">
                  <Brain className="w-8 h-8 text-on-surface-variant/30" />
                </div>
                <h3 className="text-xl font-bold text-on-surface mb-2">
                  No prediction yet
                </h3>
                <p className="text-on-surface-variant mb-6 max-w-xs">
                  Configure features and click "Generate Prediction" to see ML-powered estimates.
                </p>
                <div className="grid grid-cols-2 gap-3 text-left">
                  {featureConfigs[predictionType].map((config) => (
                    <div key={config.key} className="text-xs text-on-surface-variant/60">
                      {config.label}: <span className="font-mono text-on-surface">—</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </motion.div>
        </div>
      )}

      {activeTab === "models" && (
        <div className="space-y-6">
          {/* Metrics Overview */}
          {metrics && (
            <motion.div
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"
            >
              <MetricCard label="R² Score" value={metrics.r2.toFixed(3)} trend={metrics.r2 > 0.8 ? "+Excellent" : metrics.r2 > 0.5 ? "+Good" : undefined} icon={<Target className="w-5 h-5 text-blue-400" />} />
              <MetricCard label="RMSE" value={metrics.rmse.toFixed(2)} icon={<Activity className="w-5 h-5 text-purple-400" />} />
              <MetricCard label="MAE" value={metrics.mae.toFixed(2)} icon={<TrendingUp className="w-5 h-5 text-emerald-400" />} />
              <MetricCard label="Samples" value={metrics.samples.toFixed(0)} icon={<BarChart2 className="w-5 h-5 text-orange-400" />} />
            </motion.div>
          )}

          {/* Model Cards */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {(models.length > 0 ? models : [
              { id: "1", name: `${predictionType}_linear_regression`, version: "v1", type: predictionType, algorithm: "linear_regression", parameters: {}, metrics: metrics || { mse: 0, mae: 0, rmse: 0, r2: 0, samples: 0 }, onnx_path: "", is_active: true, created_at: Date.now(), updated_at: Date.now() },
              { id: "2", name: `${predictionType}_random_forest`, version: "v1", type: predictionType, algorithm: "random_forest", parameters: {}, metrics: metrics || { mse: 0, mae: 0, rmse: 0, r2: 0, samples: 0 }, onnx_path: "", is_active: false, created_at: Date.now(), updated_at: Date.now() },
              { id: "3", name: `${predictionType}_xgboost`, version: "v1", type: predictionType, algorithm: "xgboost", parameters: {}, metrics: metrics || { mse: 0, mae: 0, rmse: 0, r2: 0, samples: 0 }, onnx_path: "", is_active: false, created_at: Date.now(), updated_at: Date.now() },
            ]).map((model) => (
              <motion.div
                key={model.id}
                initial={{ opacity: 0, y: 16 }}
                animate={{ opacity: 1, y: 0 }}
              >
                <ModelCard
                  model={model}
                  onRetrain={() => handleRetrain(model.algorithm)}
                  onExport={handleExport}
                  onActivate={handleActivate}
                  isLoading={retrainingModel === model.algorithm || activeModel === model.id}
                />
              </motion.div>
            ))}
          </div>

          {/* Retrain Section */}
          <div className="bg-surface-container-lowest border border-outline-variant rounded-xl p-5">
            <h3 className="font-bold text-on-surface mb-4 flex items-center gap-2">
              <RotateCcw className="w-5 h-5" />
              Retrain Models
            </h3>
            <p className="text-sm text-on-surface-variant mb-4">
              Trigger retraining with different algorithms. This will collect recent build data and train new models.
            </p>
            <div className="flex flex-wrap gap-2">
              {algorithmOptions.map((algo) => (
                <button
                  key={algo.value}
                  onClick={() => handleRetrain(algo.value)}
                  disabled={retrainingModel === algo.value}
                  className={`flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold transition-all ${
                    retrainingModel === algo.value
                      ? "bg-primary-container text-on-primary-fixed-variant animate-pulse"
                      : "bg-surface-container border border-outline-variant text-on-surface hover:bg-surface-container-high"
                  }`}
                >
                  {retrainingModel === algo.value ? <Loader2 className="w-4 h-4 animate-spin" /> : <RotateCcw className="w-4 h-4" />}
                  {algo.label}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {activeTab === "history" && (
        <div className="space-y-4">
          {loadingHistory ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="bg-surface-container-lowest border border-outline-variant rounded-xl h-48 animate-pulse" />
              ))}
            </div>
          ) : history.length === 0 ? (
            <div className="bg-surface-container-lowest border border-outline-variant border-dashed rounded-xl py-24 text-center flex flex-col items-center">
              <div className="w-16 h-16 rounded-2xl bg-primary-fixed/5 border border-primary-fixed/10 flex items-center justify-center mb-5">
                <History className="w-8 h-8 text-on-surface-variant/30" />
              </div>
              <h3 className="text-xl font-bold text-on-surface mb-2">
                No prediction history
              </h3>
              <p className="text-on-surface-variant mb-6 max-w-xs">
                Predictions will appear here after you generate them from the Predict tab.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-outline-variant text-left text-xs font-semibold text-on-surface-variant uppercase tracking-wider">
                    <th className="pb-3 pr-4">Type</th>
                    <th className="pb-3 pr-4">Predicted</th>
                    <th className="pb-3 pr-4">Confidence</th>
                    <th className="pb-3 pr-4">Model</th>
                    <th className="pb-3 pr-4">Time</th>
                  </tr>
                </thead>
                <tbody>
                  {history.map((pred, idx) => (
                    <motion.tr
                      key={pred.id}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: idx * 0.03 }}
                      className="border-b border-outline-variant/50 hover:bg-surface-container/50 transition-colors"
                    >
                      <td className="py-3 pr-4">
                        <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold tracking-widest uppercase border ${typeConfig[pred.type as PredictionType]?.color || "text-gray-400 bg-gray-400/10 border-gray-400/20"}`}>
                          {typeConfig[pred.type as PredictionType]?.label || pred.type}
                        </span>
                      </td>
                      <td className="py-3 pr-4 font-mono text-on-surface">
                        {pred.output?.predicted_value?.toFixed?.(2) ?? pred.output?.output ?? "N/A"}
                      </td>
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-1.5">
                          <div className="w-16 h-1.5 bg-surface-container rounded-full overflow-hidden">
                            <div
                              className="h-full bg-primary-fixed rounded-full"
                              style={{ width: `${pred.confidence * 100}%` }}
                            />
                          </div>
                          <span className="font-semibold text-on-surface-variant">
                            {(pred.confidence * 100).toFixed(1)}%
                          </span>
                        </div>
                      </td>
                      <td className="py-3 pr-4 font-mono text-xs text-on-surface-variant">
                        {pred.model_version}
                      </td>
                      <td className="py-3 pr-4 text-on-surface-variant">
                        {formatDistanceToNow(new Date(pred.created_at), { addSuffix: true })}
                      </td>
                    </motion.tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
}