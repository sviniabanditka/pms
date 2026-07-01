# Embed + ML-сайдкар (AI): fastembed (эмбеддинги) + per-workspace классификаторы
# (Фаза 3, sklearn). ML-граница: /embed, /train, /predict. Модели — на томе /models.
import os
from typing import List, Literal, Optional

import re
import joblib
import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel
from fastembed import TextEmbedding
from sklearn.linear_model import LogisticRegression, Ridge
from sklearn.cluster import HDBSCAN
from sklearn.ensemble import IsolationForest

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


# --- Аномалии (Фаза 4): IsolationForest по эмбеддингам воркспейса ---
class AnomalyRequest(BaseModel):
    vectors: List[List[float]]


class AnomalyResponse(BaseModel):
    scores: List[float]   # 0..1, выше = аномальнее


@app.post("/anomaly", response_model=AnomalyResponse)
def anomaly(req: AnomalyRequest):
    n = len(req.vectors)
    if n < 20:
        return AnomalyResponse(scores=[0.0] * n)
    X = np.array(req.vectors, dtype=np.float32)
    iso = IsolationForest(contamination="auto", random_state=0)
    iso.fit(X)
    raw = -iso.score_samples(X)  # выше = аномальнее
    lo, hi = float(raw.min()), float(raw.max())
    span = hi - lo or 1.0
    return AnomalyResponse(scores=[float((s - lo) / span) for s in raw])


# --- Экстрактивное резюме (Фаза 4): TextRank по эмбеддингам предложений ---
class SummarizeRequest(BaseModel):
    text: str
    max_sentences: int = 3


class SummarizeResponse(BaseModel):
    summary: str


_SENT_SPLIT = re.compile(r"(?<=[.!?…])\s+|\n+")


@app.post("/summarize", response_model=SummarizeResponse)
def summarize(req: SummarizeRequest):
    sents = [s.strip() for s in _SENT_SPLIT.split(req.text or "") if len(s.strip()) > 15]
    k = max(1, req.max_sentences)
    if len(sents) <= k:
        return SummarizeResponse(summary=" ".join(sents))
    V = np.array(list(_embed.embed(sents)), dtype=np.float32)
    norms = np.linalg.norm(V, axis=1, keepdims=True)
    norms[norms == 0] = 1.0
    Vn = V / norms
    sim = Vn @ Vn.T
    np.fill_diagonal(sim, 0.0)
    sim[sim < 0] = 0.0
    # PageRank (power iteration) по графу похожести предложений
    deg = sim.sum(axis=1, keepdims=True)
    deg[deg == 0] = 1.0
    P = sim / deg
    r = np.ones(len(sents)) / len(sents)
    for _ in range(30):
        r = 0.15 / len(sents) + 0.85 * (P.T @ r)
    top = sorted(sorted(range(len(sents)), key=lambda i: r[i], reverse=True)[:k])
    return SummarizeResponse(summary=" ".join(sents[i] for i in top))


# --- Sentiment (Фаза 4): билингвальный лексикон RU+EN.
# ponytail: лексиконная эвристика, потолок ~ключевые слова; апгрейд — multilingual
# sentiment-ONNX модель, если понадобится точность на сарказме/отрицаниях.
_POS = set("""good great awesome nice excellent perfect love thanks thank helpful
works fixed solved done ready clear happy cool amazing wonderful appreciate glad win
хорошо отлично супер класс спасибо круто рад работает готово исправлено решено ясно
отличн прекрасно люблю нравится удобно быстро успех молодец топ""".split())
_NEG = set("""bad broken bug error fail failed crash slow blocked stuck confusing
wrong issue problem hate annoying terrible awful useless angry frustrated sorry cannot
плохо ошибка баг сломан падает медленно блокирует застрял непонятно неправильно
проблема ненавижу ужасно бесит бесполезно злой раздражает жаль нельзя косяк""".split())
_WORD = re.compile(r"[a-zA-Zа-яА-ЯёЁ]+")


class SentimentRequest(BaseModel):
    texts: List[str]


class SentimentResponse(BaseModel):
    scores: List[float]   # -1..1 на текст


@app.post("/sentiment", response_model=SentimentResponse)
def sentiment(req: SentimentRequest):
    out = []
    for t in req.texts:
        pos = neg = 0
        for w in _WORD.findall((t or "").lower()):
            # префиксный матч ловит словоформы (работает/работал → работает)
            if w in _POS or any(w.startswith(p) for p in _POS if len(p) > 4):
                pos += 1
            elif w in _NEG or any(w.startswith(p) for p in _NEG if len(p) > 4):
                neg += 1
        out.append((pos - neg) / (pos + neg + 1))
    return SentimentResponse(scores=out)
