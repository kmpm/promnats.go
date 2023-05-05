#!/bin/sh
set -e


if [ "$1" = 'promnats' ]; then
    
    # run as user default
    # su-exec default "$@"
    exec "$@"
fi

exec "$@"
