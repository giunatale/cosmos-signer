#!/bin/bash

for dir in */; do
    if [[ -f "${dir}build.sh" ]]; then
        pushd "${dir}" > /dev/null
        echo "Running build.sh in ${dir%/}"
        ./build.sh
	mv plugin.so "../${dir%/}.so"
        popd > /dev/null
    else
        echo "No build.sh found in ${dir%/}"
    fi
done

