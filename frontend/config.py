from pydantic import Field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    backend_url: str = Field("http://localhost:8080", env="BACKEND_URL")

    gradio_host: str = Field("0.0.0.0", env="GRADIO_HOST")
    gradio_port: int = Field(8090, env="GRADIO_PORT")
    gradio_max_file_size: str = Field("5mb", env="GRADIO_MAX_FILE_SIZE")


settings = Settings()
