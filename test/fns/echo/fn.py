#!/usr/bin/env python3

import typing

def fn(input: typing.Optional[str], headers: typing.Optional[typing.Dict[str, str]]) -> typing.Optional[str]:
    """echo the input"""
    if headers is not None:
        print ("headers: " + str(headers))

    return input
