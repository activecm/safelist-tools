#!/usr/bin/env bash

# whitelist_destination_hostnames.sh: whitelists a line delineated list of non-wildcarded hostnames as destinations
# The filename should be the sole parameter to the script.  To read from stdin, use a filename of "-"

# To set a comment to use for all additions, run this before starting the script (without the leading "#"):
#	export WL_COMMENT="Put your comment here"

# version 0.0.4
# author: Logan Lembke (hostname modifications by Bill S)


whitelist_hostname_as_destination() {
    curl -XPOST "http://${HUNTER_INTERNAL_IP}:8080/api/v0/dnscat2-ja3/whitelist/add" --data-binary @- << EOF
{
    "name": "$1",
    "type": "domain_literal",
    "comment": "$wl_comment",
    "domain": "$1"
}
EOF
}


HUNTER_INTERNAL_IP=`sudo docker inspect achunter_api -f "{{.NetworkSettings.Networks.achunter_default.IPAddress}}" 2>/dev/null`
if [ -z "$HUNTER_INTERNAL_IP" ]; then
	HUNTER_INTERNAL_IP=`sudo docker inspect achunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}" 2>/dev/null`
fi
if [ -z "$HUNTER_INTERNAL_IP" ]; then
	HUNTER_INTERNAL_IP=`sudo docker inspect aihunter_api -f "{{.NetworkSettings.Networks.aihunter_default.IPAddress}}"`
fi

if [ -z "$HUNTER_INTERNAL_IP" ]; then
    echo "Unable to locate the API Container, exiting." >&2
    exit 1
fi

wl_comment="${WL_COMMENT:-Externally added entry}"


for hostname in `cat $1`; do
    if [ -n "$hostname" ]; then
        printf "Whitelisting $hostname: "
        whitelist_hostname_as_destination "$hostname"
        printf "\n"
    fi
done
