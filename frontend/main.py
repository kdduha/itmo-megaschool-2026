import base64
import json
import logging

import gradio as gr
import requests
from sseclient import SSEClient

from config import settings

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s"
)
logger = logging.getLogger(__name__)


def explain_image_stream(prompt: gr.Text, file: gr.File, temperature: float=0.7, max_tokens: int=256):
    if file is None:
        yield "❌ No file uploaded", "Error"
        return

    filepath = file.value
    try:
        with open(filepath, "rb") as f:
            file_bytes = f.read()
        logger.info(f"Read {len(file_bytes)} bytes from file")

    except Exception as e:
        logger.error(f"Failed to read file: {e}")
        yield f"❌ Error reading file: {e}", "Error"
        return

    payload = {
        "file_base64": base64.b64encode(file_bytes).decode("utf-8"),
        "file_name": filepath.split("/")[-1],
        "file_format": filepath.split(".")[-1],
        "generation": {"temperature": temperature, "max_tokens": max_tokens},
    }

    if prompt:
        payload["prompt"] = prompt.value

    url = f"{settings.backend_url}/explain/stream"
    headers = {"Content-Type": "application/json"}

    accumulated_text = ""

    try:
        resp = requests.post(url, headers=headers, json=payload, stream=True)
        resp.raise_for_status()

        client = SSEClient(resp)

        for event in client.events():

            if event.event == "message":
                chunk = json.loads(event.data)
                accumulated_text += chunk.get("delta", "")
                yield accumulated_text, "Generating..."

            elif event.event == "done":
                chunk = json.loads(event.data)
                accumulated_text += chunk.get("explanation", "")
                yield accumulated_text, "Finalizing..."

            elif event.event == "error":
                yield f"❌ Backend error: {event.data}", "Error"
                return

        yield accumulated_text, "Success"

    except Exception as e:
        logger.exception(f"Request failed: {e}")
        yield f"❌ Request failed: {e}", "Error"


if __name__ == "__main__":

    with gr.Blocks() as demo:
        gr.Markdown("## Explain diagram from image")

        with gr.Row():
            prompt_input = gr.Textbox(
                label="Prompt", placeholder="Explain architecture"
            )
            image_input = gr.File(
                label="Upload Diagram", file_count="single", type="filepath"
            )

        with gr.Row():
            temperature_input = gr.Slider(
                minimum=0.0, maximum=1.0, value=0.7, step=0.01, label="Temperature"
            )
            max_tokens_input = gr.Slider(
                minimum=1, maximum=1024, value=512, step=1, label="Max Tokens"
            )

        status_output = gr.Textbox(show_label=False, lines=1, interactive=False, max_lines=1, value="")
        output = gr.Markdown(
            label="Explanation", buttons="copy", container=True, padding=True
        )

        btn = gr.Button("Explain")
        btn.click(
            fn=explain_image_stream,
            inputs=[prompt_input, image_input, temperature_input, max_tokens_input],
            outputs=[output, status_output],
        )

    demo.launch(
        server_name=settings.gradio_host,
        server_port=settings.gradio_port,
        max_file_size=settings.gradio_max_file_size,
        share=False,
    )
