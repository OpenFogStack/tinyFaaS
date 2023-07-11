#!/usr/bin/env python3

import typing
import kv


def fn(input: typing.Optional[str]) -> typing.Optional[str]:
    """add the input to a list, output the list"""

    try:
        l = kv.read("list")[0]
    except:
        l = ""

    l = l + input + "\n"

    kv.update("list", l)

    return str(l)
