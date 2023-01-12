#!/usr/bin/env python3

import http.server
import socketserver
import subprocess

if __name__ == "__main__":

    # create a webserver at port 8080 and execute fn.sh for every request
    class tinyFaaSFNHandler(http.server.BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            print(f"GET {self.path}")
            if self.path == "/health":
                self.send_response(200)
                self.end_headers()
                self.wfile.write("OK".encode("utf-8"))
                print("reporting health: OK")
                return

            self.send_response(404)
            self.end_headers()
            return

        def do_POST(self) -> None:
            d = self.rfile.read(int(self.headers["Content-Length"])).decode("utf-8")
            try:
                res = subprocess.run(["./fn.sh"], input=d.encode("utf-8"), check=True, capture_output=True)
                self.send_response(200)
                self.end_headers()
                self.wfile.write(res.stdout)
                return
            except Exception as e:
                self.send_response(500)
                self.end_headers()
                self.wfile.write(str(e).encode("utf-8"))
                return

    with socketserver.TCPServer(("", 8000), tinyFaaSFNHandler) as httpd:
        httpd.serve_forever()
