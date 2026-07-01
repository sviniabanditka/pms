# Embed + ML-сайдкар (AI): fastembed (эмбеддинги) + per-workspace классификаторы
# (Фаза 3, sklearn). ML-граница: /embed, /train, /predict. Модели — на томе /models.
import os
from typing import List, Literal, Optional

import joblib
import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel
from fastembed import TextEmbedding
from sklearn.linear_model import LogisticRegression, Ridge
from sklearn.cluster import HDBSCAN

MODEL = os.environ.get("EMBED_MODEL", "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2")
MODELS_DIR = os.environ.get("MODELS_DIR", "/models")
os.makedirs(MODELS_DIR, exist_ok=True)
_embed = TextEmbedding(model_name=MODEL)
_clf_cache: dict = {}  # workspace_id → (mtime, clf)

app = FastAPI()


def _model_path(ws: str, kind: str) -> str:
    safe = "".join(c for c in ws if c.isalnum() or c in "-_")
    return os.path.join(MODELS_DIR, f"{safe}.{kind}.joblib")


@app.get("/health")
def health():
    return {"ok": True, "model": MODEL}


class EmbedRequest(BaseModel):
    texts: List[str]
    kind: Literal["query", "passage"] = "passage"


class EmbedResponse(BaseModel):
    vectors: List[List[float]]


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    if "e5" in MODEL.lower():
        prefix = "query: " if req.kind == "query" else "passage: "
        texts = [prefix + (t or "") for t in req.texts]
    else:
        texts = [t or "" for t in req.texts]
    return EmbedResponse(vectors=[v.tolist() for v in _embed.embed(texts)])


class TrainRequest(BaseModel):
    workspace_id: str
    kind: str = "priority"                 # priority | estimate | slip
    task: Literal["classification", "regression"] = "classification"
    vectors: List[List[float]]
    labels: List[float]
    weights: Optional[List[float]] = None   # sample_weight (петля фидбэка)


@app.post("/train")
def train(req: TrainRequest):
    ys = req.labels
    X = np.array(req.vectors, dtype=np.float32)
    if req.task == "regression":
        if len(ys) < 40:
            return {"ok": False, "reason": "insufficient", "n": len(ys)}
        model = Ridge(alpha=1.0)
        sw = np.array(req.weights, dtype=np.float32) if req.weights else None
        model.fit(X, np.array(ys, dtype=np.float32), sample_weight=sw)
    else:
        if len(ys) < 40 or len(set(ys)) < 2:
            return {"ok": False, "reason": "insufficient", "n": len(ys), "classes": len(set(ys))}
        model = LogisticRegression(max_iter=1000, class_weight="balanced")
        sw = np.array(req.weights, dtype=np.float32) if req.weights else None
        model.fit(X, np.array(ys).astype(int), sample_weight=sw)
    path = _model_path(req.workspace_id, req.kind)
    joblib.dump({"model": model, "task": req.task}, path)
    _clf_cache.pop(path, None)
    return {"ok": True, "n": len(ys), "task": req.task}


class PredictRequest(BaseModel):
    workspace_id: str
    kind: str = "priority"
    vectors: List[List[float]]


class PredictResponse(BaseModel):
    predictions: List[float]
    confidence: List[float]


@app.post("/predict", response_model=PredictResponse)
def predict(req: PredictRequest):
    path = _model_path(req.workspace_id, req.kind)
    if not os.path.exists(path):
        return PredictResponse(predictions=[], confidence=[])
    mtime = os.path.getmtime(path)
    cached = _clf_cache.get(path)
    if not cached or cached[0] != mtime:
        cached = (mtime, joblib.load(path))
        _clf_cache[path] = cached
    bundle = cached[1]
    model, task = bundle["model"], bundle["task"]
    X = np.array(req.vectors, dtype=np.float32)
    preds = model.predict(X)
    if task == "regression":
        return PredictResponse(predictions=[float(p) for p in preds], confidence=[1.0] * len(preds))
    probs = model.predict_proba(X)
    conf = probs.max(axis=1)
    return PredictResponse(predictions=[float(p) for p in preds], confidence=[float(c) for c in conf])


class ClusterRequest(BaseModel):
    vectors: List[List[float]]
    min_cluster_size: int = 8


class ClusterResponse(BaseModel):
    labels: List[int]   # cluster id на каждый вектор (-1 = шум)


@app.post("/cluster", response_model=ClusterResponse)
def cluster(req: ClusterRequest):
    n = len(req.vectors)
    mcs = max(3, req.min_cluster_size)
    if n < mcs * 2:
        return ClusterResponse(labels=[-1] * n)
    X = np.array(req.vectors, dtype=np.float32)
    # нормируем → euclidean ≈ cosine
    norms = np.linalg.norm(X, axis=1, keepdims=True)
    norms[norms == 0] = 1.0
    Xn = X / norms
    clu = HDBSCAN(min_cluster_size=mcs, metric="euclidean")
    labels = clu.fit_predict(Xn)
    return ClusterResponse(labels=[int(l) for l in labels])
