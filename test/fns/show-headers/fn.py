import json
import typing

def fn(input: typing.Optional[str], headers: typing.Optional[typing.Dict[str, str]]) -> typing.Optional[str]:
    """echo the input"""
    if headers is not None:
        return json.dumps(headers)
    else:
        return "{}"