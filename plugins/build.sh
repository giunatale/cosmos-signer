#!/bin/bash

for dir in */; do
    if [[ -f "${dir}build.sh" ]]; then
        pushd "${dir}" > /dev/null
        echo "Building ${dir%/} plugin"
        ./build.sh
        mkdir -p "../../build/plugins"
        mv plugin.so "../../build/plugins/${dir%/}.so"
        popd > /dev/null
    else
        echo "No build.sh found in ${dir%/}"
    fi
done

