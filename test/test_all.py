#!/usr/bin/env python3

import unittest

import os
import subprocess
import sys
import typing
import urllib.error
import urllib.request

connection: typing.Dict[str, typing.Union[str, int]] = {}

def setUpModule() -> None:
    """start tinyfaas instance"""

    # call make clean
    try:
        subprocess.run(["make", "clean"], cwd="..", check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to clean up:\n{e.stderr.decode('utf-8')}")

    # call make prepare
    try:
        subprocess.run(["make", "prepare"], cwd="..", check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to prepare:\n{e.stderr.decode('utf-8')}")

    # call make start
    try:
        subprocess.run(["make", "start"], cwd="..", check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to start:\n{e.stderr.decode('utf-8')}")

    # set up connection
    global connection
    connection["host"] = "localhost"
    connection["management_port"] = 8080
    connection["http_port"] = 80
    connection["grpc_port"] = 8000
    connection["coap_port"] = 5683

    # wait for tinyfaas to start
    while True:
        try:
            urllib.request.urlopen(f"http://{connection['host']}:{connection['management_port']}/")
            break
        except urllib.error.HTTPError:
            break
        except Exception as e:
            continue
    # wait for tinyfaas to start
    while True:
        try:
            urllib.request.urlopen(f"http://{connection['host']}:{connection['http_port']}/")
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
        subprocess.run(["./wipe-functions.sh"], cwd="../scripts", check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to wipe functions:\n{e.stderr.decode('utf-8')}")

    # call make clean
    try:
        subprocess.run(["make", "clean"], cwd="..", check=True, capture_output=True)
    except subprocess.CalledProcessError as e:
        print(f"Failed to clean up:\n{e.stderr.decode('utf-8')}")

    return

def startFunction(folder_name: str, fn_name: str, env: str, threads: int) -> str:
    """starts a function, returns name"""

    # get full path of folder
    folder_name = os.path.abspath(folder_name)

    # use the upload.sh script
    try:
        subprocess.run(["./upload.sh", folder_name, fn_name, env, str(threads)], cwd="../scripts", check=True, capture_output=True)
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
        cls.fn = startFunction("./fns/sieve-of-erasthostenes", "sieve", "nodejs", 1)

    def setUp(self) -> None:
        super(TestSieve, self).setUp()
        self.fn = TestSieve.fn

    def test_invoke(self) -> None:
        """invoke a function"""

        # make a request to the function
        res = urllib.request.urlopen(f"http://{self.host}:{self.http_port}/{self.fn}", timeout=10)

        # check the response
        self.assertEqual(res.status, 200)

        return

    def test_invoke_async(self) -> None:
        """invoke a function async"""

        # make an async request to the function
        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            headers={'X-tinyFaaS-Async': 'true'}
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 202)

        return

    def test_invoke_grpc(self) -> None:
        """invoke a function"""
        pass

    def test_invoke_coap(self) -> None:
        """invoke a function with CoAP"""

        try:
            import asyncio
            import aiocoap
        except ImportError:
            self.skipTest("aiocoap is not installed -- if you want to run CoAP tests, install the dependencies in requirements.txt")
            return

        msg = aiocoap.Message(code=aiocoap.GET, uri=f"coap://{self.host}:{self.coap_port}/{self.fn}")

        async def main() -> aiocoap.Message:
            protocol = await aiocoap.Context.create_client_context()
            response = await protocol.request(msg).response
            return response

        response = asyncio.run(main())

        self.assertIsNotNone(response)
        self.assertEqual(response.code, aiocoap.CONTENT)

        return

class TestEcho(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        super(TestEcho, cls).setUpClass()
        cls.fn = startFunction("./fns/echo", "echo", "python3", 1)

    def setUp(self) -> None:
        super(TestEcho, self).setUp()
        self.fn = TestEcho.fn

    def test_invoke(self) -> None:
        """invoke a function"""

        # make a request to the function with a payload
        payload = "Hello World!"

        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            data=payload.encode('utf-8'),
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 200)
        self.assertEqual(res.read().decode('utf-8'), payload)

        return

class TestBinary(TinyFaaSTest):
    fn = ""

    @classmethod
    def setUpClass(cls) -> None:
        super(TestBinary, cls).setUpClass()
        cls.fn = startFunction("./fns/echo-binary", "echo-binary", "binary", 1)

    def setUp(self) -> None:
        super(TestBinary, self).setUp()
        self.fn = TestBinary.fn

    def test_invoke(self) -> None:
        """invoke a function"""

        # make a request to the function with a payload
        payload = "Hello World!"

        req = urllib.request.Request(
            f"http://{self.host}:{self.http_port}/{self.fn}",
            data=payload.encode('utf-8'),
        )

        res = urllib.request.urlopen(req, timeout=10)

        # check the response
        self.assertEqual(res.status, 200)
        self.assertEqual(res.read().decode('utf-8'), payload)

        return

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

    unittest.main() # run all tests
