# Embed-сайдкар: fastembed (ONNX, без torch). Одна модель на весь SaaS, stateless.
# e5-модели требуют префиксов: "query: " для запросов, "passage: " для документов.
import os
from typing import List, Literal

from fastapi import FastAPI
from pydantic import BaseModel
from fastembed import TextEmbedding

MODEL = os.environ.get("EMBED_MODEL", "intfloat/multilingual-e5-small")
_model = TextEmbedding(model_name=MODEL)

app = FastAPI()


class EmbedRequest(BaseModel):
    texts: List[str]
    kind: Literal["query", "passage"] = "passage"


class EmbedResponse(BaseModel):
    vectors: List[List[float]]


@app.get("/health")
def health():
    return {"ok": True, "model": MODEL}


@app.post("/embed", response_model=EmbedResponse)
def embed(req: EmbedRequest):
    # e5-модели требуют query:/passage: префиксы; остальным (paraphrase-multilingual) — нет
    if "e5" in MODEL.lower():
        prefix = "query: " if req.kind == "query" else "passage: "
        texts = [prefix + (t or "") for t in req.texts]
    else:
        texts = [t or "" for t in req.texts]
    vectors = [v.tolist() for v in _model.embed(texts)]
    return EmbedResponse(vectors=vectors)
