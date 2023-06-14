# tinyFaaS Tests

This directory contains tests for tinyFaaS.
These tests use Python3 and the `unittest` package, which is part of the Python3
standard library.

Further, these tests start a local tinyFaaS instance, assuming no instance is
already running.
This requires `make` and Docker to be installed.

Additional Python3 packages are necessary for some tests.
These can be installed with `python3 -m pip install -r requirements.txt`.
Alternatively, you can also use a virtual environment.
If these packages are not installed, some tests may be skipped.

Run these tests with:

```sh
python3 test_all.py
```
