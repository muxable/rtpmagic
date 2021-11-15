#!/bin/bash

docker run -t skandyla/fpm -s dir -t deb ${FPM_OPTS} -f \
    -maintainer "$PKG_MAINTAINER" \
    --vendor "$PKG_VENDOR" \
    --url "$PKG_URL" \
    --description "$PKG_DESCRIPTION" \
    --architecture "amd64" \
    -C /tmp/proj \
    .