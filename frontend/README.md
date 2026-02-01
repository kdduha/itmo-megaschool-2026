# Frontend

Here you can find a python frontend service with basic [`gradio`](https://www.gradio.app/) UI, that:
- sends diagram files to the [backend](../backend/README.md) service
- streams backend's responses

## Local Setup

You can check the basic frontend configuration and available params [here](./config.py)
The project is managed by [`uv`](https://docs.astral.sh/uv/)

1. Sync dependencies
    ```sh
    uv sync 
    ```
1. Start the server:
    ```sh
    uv run main.py
    ```

## Docker Setup

You can set envs in order to configure the server

1. Build an image:
   ```sh
   docker build -t py-frontend .
   ```
1. Start the server:
   ```sh
   docker run -it --rm py-frontend
   ```

## Developing

Some useful commands:
- Code formatting
  ```sh
  uvx isort . && uvx black . 
  ```
