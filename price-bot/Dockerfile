# Worker de coleta: consome a fila bot.collect (RabbitMQ), abre o site com o
# Chromium do Playwright, interpreta com o Gemini e grava no price-back.
FROM python:3.13-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 PYTHONIOENCODING=utf-8 \
    PIP_ROOT_USER_ACTION=ignore PIP_DISABLE_PIP_VERSION_CHECK=1
WORKDIR /app

# libatomic1: exigida pelo Node embutido no driver do Playwright.
RUN apt-get update -o Acquire::Retries=5 \
 && apt-get install -y --no-install-recommends -o Acquire::Retries=5 libatomic1 \
 && rm -rf /var/lib/apt/lists/*

COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .

# Chromium e as bibliotecas de sistema que ele usa (camada separada para
# reaproveitar o cache das anteriores se o download falhar).
RUN apt-get update -o Acquire::Retries=5 \
 && playwright install --with-deps chromium \
 && rm -rf /var/lib/apt/lists/*

COPY . .
CMD ["python", "main.py", "worker"]
