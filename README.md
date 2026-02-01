# itmo-megaschool-2026

norm descr TBD later

# Для CPU
docker-compose -f compose.cpu.yaml up --build

# Для GPU
docker-compose -f compose.gpu.yaml up --build

# Для локалки (без LLM)
docker-compose -f compose.local.yaml up --build
