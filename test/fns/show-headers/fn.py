#!/usr/bin/env python3

import typing

def fn(input: typing.Optional[str], headers: typing.Optional[typing.Dict[str, str]]) -> typing.Optional[str]:
    """echo the input"""
    msg = ""
    if headers is not None:
        msg = str(headers)
    else:
        msg = "{}"


    return msg
