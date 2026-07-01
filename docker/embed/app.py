# Embed + ML-сайдкар (AI): fastembed (эмбеддинги) + per-workspace классификаторы
# (Фаза 3, sklearn). ML-граница: /embed, /train, /predict. Модели — на томе /models.
import os
from typing import List, Literal, Optional

import joblib
import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel
from fastembed import TextEmbedding
from sklearn.linear_model import LogisticRegression

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
    kind: str = "priority"        # какой таргет учим
    vectors: List[List[float]]
    labels: List[int]


@app.post("/train")
def train(req: TrainRequest):
    ys = req.labels
    if len(ys) < 40 or len(set(ys)) < 2:
        return {"ok": False, "reason": "insufficient data", "n": len(ys), "classes": len(set(ys))}
    clf = LogisticRegression(max_iter=1000, class_weight="balanced")
    clf.fit(np.array(req.vectors, dtype=np.float32), np.array(ys))
    path = _model_path(req.workspace_id, req.kind)
    joblib.dump(clf, path)
    _clf_cache.pop(path, None)
    return {"ok": True, "n": len(ys), "classes": sorted(set(int(y) for y in ys))}


class PredictRequest(BaseModel):
    workspace_id: str
    kind: str = "priority"
    vectors: List[List[float]]


class PredictResponse(BaseModel):
    predictions: List[int]
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
    clf = cached[1]
    X = np.array(req.vectors, dtype=np.float32)
    preds = clf.predict(X)
    probs = clf.predict_proba(X)
    conf = probs.max(axis=1)
    return PredictResponse(predictions=[int(p) for p in preds], confidence=[float(c) for c in conf])
