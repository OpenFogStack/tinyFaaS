#!/usr/bin/env python3

import http.server
import socketserver

if __name__ == "__main__":
    try:
        import fn
    except ImportError:
        raise ImportError("Failed to import fn.py")

    # create a webserver at port 8080 and execute fn.fn for every request
    class tinyFaaSFNHandler(http.server.BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            print(f"GET {self.path}")
            if self.path == "/health":
                self.send_response(200)
                self.end_headers()
                self.wfile.write("OK".encode("utf-8"))
                print("reporting health: OK")
                return

            try:
                _ = fn.fn()
                self.send_response(200)
                self.end_headers()
                return
            except Exception as e:
                print(e)
                self.send_response(500)
                self.end_headers()
                return

        def do_POST(self) -> None:
            d = self.rfile.read(int(self.headers["Content-Length"]))
            try:
                res = fn.fn(d)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(res)
                return
            except Exception as e:
                print(e)
                self.send_response(500)
                self.end_headers()
                self.wfile.write(str(e).encode("utf-8"))
                return

    with socketserver.TCPServer(("", 8000), tinyFaaSFNHandler) as httpd:
        httpd.serve_forever()
