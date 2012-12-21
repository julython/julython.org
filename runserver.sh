#!/bin/bash
#
#  Simple gunicorn script for running the site
#

set -e
LOGFILE=/var/log/gunicorn/julython.log
LOGDIR=$(dirname $LOGFILE)
NUM_WORKERS=3
USER=www-data
GROUP=www-data
source /usr/local/julython/venv/bin/activate
cd /usr/local/julython/venv/src/july
test -d $LOGDIR || mkdir -p $LOGDIR
export NEW_RELIC_CONFIG_FILE=/etc/newrelic/newrelic.conf
exec newrelic-admin run-program gunicorn_django -w $NUM_WORKERS \
    --user=$USER --group=$GROUP --log-level=info \
    --log-file=$LOGFILE 2>>$LOGFILE