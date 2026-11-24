#!/usr/bin/env python3
"""mock_upstream.py —— 冒烟测试用的假大模型上游（OpenAI 兼容）。
用法：python3 mock_upstream.py [port]
"""
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 9999


class H(BaseHTTPRequestHandler):
    def _json(self, code, obj):
        b = json.dumps(obj, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_GET(self):
        if "/models" in self.path:
            self._json(200, {"object": "list", "data": [{"id": "test-model", "object": "model"}, {"id": "gpt-4o", "object": "model"}]})
        else:
            self._json(404, {"error": {"message": "not found"}})

    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        if self.path.endswith("/chat/completions"):
            if req.get("stream"):
                self._stream(req)
            else:
                self._json(200, self._resp(req))
        else:
            self._json(404, {"error": {"message": "not found"}})

    def _resp(self, req):
        return {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "created": 1700000000,
            "model": req.get("model", "test-model"),
            "choices": [{"index": 0, "message": {"role": "assistant", "content": "你好，这是 mock 回复"}, "finish_reason": "stop"}],
            "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30},
        }

    def _stream(self, req):
        model = req.get("model", "test-model")
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        chunks = [
            {"id": "chatcmpl-mock", "object": "chat.completion.chunk", "created": 1700000000, "model": model,
             "choices": [{"index": 0, "delta": {"role": "assistant", "content": ""}, "finish_reason": None}]},
            {"id": "chatcmpl-mock", "object": "chat.completion.chunk", "created": 1700000000, "model": model,
             "choices": [{"index": 0, "delta": {"content": "你好"}, "finish_reason": None}]},
            {"id": "chatcmpl-mock", "object": "chat.completion.chunk", "created": 1700000000, "model": model,
             "choices": [], "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}},
        ]
        for c in chunks:
            self.wfile.write(b"data: " + json.dumps(c, ensure_ascii=False).encode() + b"\n\n")
        self.wfile.write(b"data: [DONE]\n\n")

    def log_message(self, *args):
        pass


HTTPServer(("127.0.0.1", PORT), H).serve_forever()