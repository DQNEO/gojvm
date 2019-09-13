#!/usr/bin/env bash

set -e

actual=$(cat HelloWorld.class | ./gojvm)
expected="Hello world"
if [[ $actual == $expected ]];then
    echo "ok"
else
    echo "not ok"
fi

actual=$(cat Arith.class | ./gojvm)
expected="42"
if [[ $actual == $expected ]];then
    echo "ok"
else
    echo "not ok"
fi

echo "All tests passed."
