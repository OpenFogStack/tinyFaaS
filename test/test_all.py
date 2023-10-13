#!/usr/bin/env python3

import unittest

import os
import os.path as path
import signal
import subprocess
import sys
import typing
import urllib.error
import urllib.request

connection: typing.Dict[str, typing.Union[str, int]] = {
    "host": "localhost",
    "management_port": 8080,
    "http_port": 8000,
    "grpc_port": 9000,
    "coap_port": 5683,
}

tf_process: typing.Optional[subprocess.Popen] = None  # type: ignore
src_path = "."
fn_path = path.join(src_path, "test", "fns")
script_path = path.join(src_path, "scripts")
grpc_api_path = path.join(src_path, "pkg", "grpc", "tinyfaas")


def setUpModule() -> None:
    """start tinyfaas instance"""
    # call make clean
    try:
        subprocess.run(["make", "clean"], cwd=src_path, check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to clean up:\n{e.stderr.decode('utf-8')}")

    # start tinyfaas
    try:
        env = os.environ.copy()
        env["HTTP_PORT"] = str(connection["http_port"])
        env["GRPC_PORT"] = str(connection["grpc_port"])
        env["COAP_PORT"] = str(connection["coap_port"])

        global tf_process

        # os.makedirs(path.join(src_path, "tmp"), exist_ok=True)
        with open(path.join(".", "tf_test.out"), "w") as f:
            tf_process = subprocess.Popen(
                ["./manager"],
                cwd=src_path,
                env=env,
                stdout=f,
                stderr=f,
            )

    except subprocess.CalledProcessError as e:
        print(f"Failed to start:\n{e.stderr.decode('utf-8')}")

    # wait for tinyfaas to start
    while True:
        try:
            urllib.request.urlopen(
                f"http://{connection['host']}:{connection['management_port']}/"
            )
            break
        except urllib.error.HTTPError:
            break
        except Exception as e:
            continue
    # wait for tinyfaas to start
    while True:
        try:
            urllib.request.urlopen(
                f"http://{connection['host']}:{connection['http_port']}/"
            )
            break
        except urllib.error.HTTPError:
            break
        except Exception as e:
            continue

    return


def tearDownModule() -> None:
    """stop tinyfaas instance"""

    # call wipe-functions.sh
    try:
        subprocess.run(
            ["./wipe-functions.sh"], cwd=script_path, check=True, capture_output=True
        )
    except subprocess.CalledProcessError as e:
        print(f"Failed to wipe functions:\n{e.stderr.decode('utf-8')}")

    # stop tinyfaas
    # with open(path.join(src_path, "tmp", "tf_test.out"), "w") as f:
    #     f.write(tf_process.stdout.read())
    #     f.write(tf_process.stderr.read())

    try:
        tf_process.send_signal(signal.SIGINT)  # type: ignore
        tf_process.wait(timeout=1)  # type: ignore
        tf_process.terminate()  # type: ignore
    except subprocess.CalledProcessError as e:
        print(f"Failed to stop:\n{e.stderr.decode('utf-8')}")
    except subprocess.TimeoutExpired:
        print("Failed to stop: Timeout expired")

    # call make clean
    try:
        subprocess.run(["make", "clean"], cwd=src_path, check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to clean up:\n{e.stderr.decode('utf-8')}")

    return


def startFunction(folder_name: str, fn_name: str, env: str, threads: int) -> str:
    """starts a function, returns name"""

    # get full path of folder
    folder_name = os.path.abspath(folder_name)

    # use the upload.sh script
    try:
        subprocess.run(
            ["./upload.sh", folder_name, fn_name, env, str(threads)],
            cwd=script_path,
            check=True,
            capture_output=True,
        )
    except subprocess.CalledProcessError as e:
        print(f"Failed to upload function {fn_name}:\n{e.stderr.decode('utf-8')}")
        raise e

    return fn_name


class TinyFaaSTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        super(TinyFaaSTest, cls).setUpClass()

    def setUp(self) -> None:
        global connection
        self.host = connection["host"]
        self.http_port = connection["http_port"]
        self.grpc_port = connection["grpc_port"]
        self.coap_port = connection["coap_port"]


class TestSieve(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        cls.fn = startFunction(
            path.join(fn_path, "sieve-of-eratosthenes"), "sieve", "nodejs", 1
        )

    def setUp(self) -> None:
        super(TestSieve, self).setUp()
        self.fn = TestSieve.fn

    def test_invoke_http(self) -> None:
        """invoke a function"""

        # make a request to the function
        res = urllib.request.urlopen(
            f"http://{self.host}:{self.http_port}/{self.fn}", timeout=10
        )

        # check the response
        self.assertEqual(res.status, 200)

        return

    def test_invoke_http_async(self) -> None:
        """invoke a function async"""

        # make an async request to the function
        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            headers={"X-tinyFaaS-Async": "true"},
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 202)

        return

    def test_invoke_coap(self) -> None:
        """invoke a function with CoAP"""

        try:
            import asyncio
            import aiocoap
        except ImportError:
            self.skipTest(
                "aiocoap is not installed -- if you want to run CoAP tests, install the dependencies in requirements.txt"
            )
            return

        msg = aiocoap.Message(
            code=aiocoap.GET, uri=f"coap://{self.host}:{self.coap_port}/{self.fn}"
        )

        async def main() -> aiocoap.Message:
            protocol = await aiocoap.Context.create_client_context()
            response = await protocol.request(msg).response
            await protocol.shutdown()
            return response

        response = asyncio.run(main())

        self.assertIsNotNone(response)
        self.assertEqual(response.code, aiocoap.CONTENT)

        return

    def test_invoke_grpc(self) -> None:
        """invoke a function"""
        try:
            import grpc
        except ImportError:
            self.skipTest(
                "grpc is not installed -- if you want to run gRPC tests, install the dependencies in requirements.txt"
            )
            return

        sys.path.append(grpc_api_path)

        import tinyfaas_pb2
        import tinyfaas_pb2_grpc

        with grpc.insecure_channel(f"{self.host}:{self.grpc_port}") as channel:
            stub = tinyfaas_pb2_grpc.TinyFaaSStub(channel)
            response = stub.Request(tinyfaas_pb2.Data(functionIdentifier=self.fn))

        self.assertIsNotNone(response)
        self.assertIsNot(response.response, "")


class TestEcho(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        super(TestEcho, cls).setUpClass()
        cls.fn = startFunction(path.join(fn_path, "echo"), "echo", "python3", 1)

    def setUp(self) -> None:
        super(TestEcho, self).setUp()
        self.fn = TestEcho.fn

    def test_invoke_http(self) -> None:
        """invoke a function"""

        # make a request to the function with a payload
        payload = "Hello World!"

        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            data=payload.encode("utf-8"),
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 200)
        self.assertEqual(res.read().decode("utf-8"), payload)

        return

    def test_invoke_coap(self) -> None:
        """invoke a function with CoAP"""

        try:
            import asyncio
            import aiocoap
        except ImportError:
            self.skipTest(
                "aiocoap is not installed -- if you want to run CoAP tests, install the dependencies in requirements.txt"
            )
            return

        # make a request to the function with a payload
        payload = "Hello World!"

        msg = aiocoap.Message(
            code=aiocoap.GET,
            uri=f"coap://{self.host}:{self.coap_port}/{self.fn}",
            payload=payload.encode("utf-8"),
        )

        async def main() -> aiocoap.Message:
            protocol = await aiocoap.Context.create_client_context()
            response = await protocol.request(msg).response
            await protocol.shutdown()
            return response

        response = asyncio.run(main())

        self.assertIsNotNone(response)
        self.assertEqual(response.code, aiocoap.CONTENT)
        self.assertEqual(response.payload.decode("utf-8"), payload)

        return

    def test_invoke_grpc(self) -> None:
        """invoke a function"""
        try:
            import grpc
        except ImportError:
            self.skipTest(
                "grpc is not installed -- if you want to run gRPC tests, install the dependencies in requirements.txt"
            )
            return

        sys.path.append(grpc_api_path)

        import tinyfaas_pb2
        import tinyfaas_pb2_grpc

        # make a request to the function with a payload
        payload = "Hello World!"

        with grpc.insecure_channel(f"{self.host}:{self.grpc_port}") as channel:
            stub = tinyfaas_pb2_grpc.TinyFaaSStub(channel)
            response = stub.Request(
                tinyfaas_pb2.Data(functionIdentifier=self.fn, data=payload)
            )

        self.assertIsNotNone(response)
        self.assertEqual(response.response, payload)


class TestEchoJS(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        super(TestEchoJS, cls).setUpClass()
        cls.fn = startFunction(path.join(fn_path, "echo-js"), "echojs", "nodejs", 1)

    def setUp(self) -> None:
        super(TestEchoJS, self).setUp()
        self.fn = TestEchoJS.fn

    def test_invoke_http(self) -> None:
        """invoke a function"""

        # make a request to the function with a payload
        payload = "Hello World!"

        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            data=payload.encode("utf-8"),
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 200)
        self.assertEqual(res.read().decode("utf-8"), payload)

        return

    def test_invoke_coap(self) -> None:
        """invoke a function with CoAP"""

        try:
            import asyncio
            import aiocoap
        except ImportError:
            self.skipTest(
                "aiocoap is not installed -- if you want to run CoAP tests, install the dependencies in requirements.txt"
            )
            return

        # make a request to the function with a payload
        payload = "Hello World!"

        msg = aiocoap.Message(
            code=aiocoap.GET,
            uri=f"coap://{self.host}:{self.coap_port}/{self.fn}",
            payload=payload.encode("utf-8"),
        )

        async def main() -> aiocoap.Message:
            protocol = await aiocoap.Context.create_client_context()
            response = await protocol.request(msg).response
            await protocol.shutdown()
            return response

        response = asyncio.run(main())

        self.assertIsNotNone(response)
        self.assertEqual(response.code, aiocoap.CONTENT)
        self.assertEqual(response.payload.decode("utf-8"), payload)

        return

    def test_invoke_grpc(self) -> None:
        """invoke a function"""
        try:
            import grpc
        except ImportError:
            self.skipTest(
                "grpc is not installed -- if you want to run gRPC tests, install the dependencies in requirements.txt"
            )
            return

        sys.path.append(grpc_api_path)

        import tinyfaas_pb2
        import tinyfaas_pb2_grpc

        # make a request to the function with a payload
        payload = "Hello World!"

        with grpc.insecure_channel(f"{self.host}:{self.grpc_port}") as channel:
            stub = tinyfaas_pb2_grpc.TinyFaaSStub(channel)
            response = stub.Request(
                tinyfaas_pb2.Data(functionIdentifier=self.fn, data=payload)
            )

        self.assertIsNotNone(response)
        self.assertEqual(response.response, payload)


class TestBinary(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        super(TestBinary, cls).setUpClass()
        cls.fn = startFunction(
            path.join(fn_path, "echo-binary"), "echobinary", "binary", 1
        )

    def setUp(self) -> None:
        super(TestBinary, self).setUp()
        self.fn = TestBinary.fn

    def test_invoke_http(self) -> None:
        """invoke a function"""

        # make a request to the function with a payload
        payload = "Hello World!"

        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            data=payload.encode("utf-8"),
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 200)
        self.assertEqual(res.read().decode("utf-8"), payload)

        return

    def test_invoke_coap(self) -> None:
        """invoke a function with CoAP"""

        try:
            import asyncio
            import aiocoap
        except ImportError:
            self.skipTest(
                "aiocoap is not installed -- if you want to run CoAP tests, install the dependencies in requirements.txt"
            )
            return

        # make a request to the function with a payload
        payload = "Hello World!"

        msg = aiocoap.Message(
            code=aiocoap.GET,
            uri=f"coap://{self.host}:{self.coap_port}/{self.fn}",
            payload=payload.encode("utf-8"),
        )

        async def main() -> aiocoap.Message:
            protocol = await aiocoap.Context.create_client_context()
            response = await protocol.request(msg).response
            await protocol.shutdown()
            return response

        response = asyncio.run(main())

        self.assertIsNotNone(response)
        self.assertEqual(response.code, aiocoap.CONTENT)
        self.assertEqual(response.payload.decode("utf-8"), payload)

        return

    def test_invoke_grpc(self) -> None:
        """invoke a function"""
        try:
            import grpc
        except ImportError:
            self.skipTest(
                "grpc is not installed -- if you want to run gRPC tests, install the dependencies in requirements.txt"
            )
            return

        sys.path.append(grpc_api_path)

        import tinyfaas_pb2
        import tinyfaas_pb2_grpc

        # make a request to the function with a payload
        payload = "Hello World!"

        with grpc.insecure_channel(f"{self.host}:{self.grpc_port}") as channel:
            stub = tinyfaas_pb2_grpc.TinyFaaSStub(channel)
            response = stub.Request(
                tinyfaas_pb2.Data(functionIdentifier=self.fn, data=payload)
            )

        self.assertIsNotNone(response)
        self.assertEqual(response.response, payload)


if __name__ == "__main__":
    # check that make is installed
    try:
        subprocess.run(["make", "--version"], check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Make is not installed:\n{e.stderr.decode('utf-8')}")
        sys.exit(1)

    # check that Docker is working
    try:
        subprocess.run(["docker", "ps"], check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Docker is not installed or not working:\n{e.stderr.decode('utf-8')}")
        sys.exit(1)

    unittest.main()  # run all tests
