import os
import uvicorn
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from sentence_transformers import SentenceTransformer

app = FastAPI(title="MPIT 2026 ML Service")

# Load model (lightweight, fast, multilingual)
# 384 dimensions
MODEL_NAME = 'paraphrase-multilingual-MiniLM-L12-v2'
print(f"Loading model {MODEL_NAME}...")
model = SentenceTransformer(MODEL_NAME)
print("Model loaded!")

class EmbeddingRequest(BaseModel):
    text: str

@app.post("/embed")
async def create_embedding(req: EmbeddingRequest):
    if not req.text:
        return {"vector": [0.0] * 384}
    
    try:
        # Generate embedding
        embedding = model.encode(req.text)
        return {"vector": embedding.tolist()}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/health")
async def health():
    return {"status": "ok", "model": MODEL_NAME}

if __name__ == "__main__":
    port = int(os.getenv("PORT", 5000))
    uvicorn.run(app, host="0.0.0.0", port=port)
