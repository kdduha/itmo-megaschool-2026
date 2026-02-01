FROM python:3.11-slim-trixie

RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates

ADD https://astral.sh/uv/install.sh /uv-installer.sh
RUN sh /uv-installer.sh && rm /uv-installer.sh

ENV PATH="/root/.local/bin/:$PATH"

WORKDIR /app
COPY . .

RUN uv sync 

CMD ["uv", "run", "main.py"]
